package sheets

import (
	"fmt"

	"google.golang.org/api/sheets/v4"
)

const (
	fieldMaskTextFormatRuns = "textFormatRuns"
)

// buildFormatRequests creates the Google Sheets API batch update requests
// to apply theme styling, frozen panes, zebra banding, and column dimensions.
// If spec.SkipFormatting is true, it returns nil immediately without issuing formatting requests.
func buildFormatRequests(sheetID int64, existingBandedRangeIDs []int64, spec TabSpec, totalRows, totalCols int64) []*sheets.Request {
	if spec.SkipFormatting {
		return nil
	}

	var reqs []*sheets.Request

	theme := spec.Theme
	if theme == nil {
		theme = ThemeCorporateNavy()
	}

	// 0. Only delete prior banding ranges if banding is enabled in the new theme.
	if theme.EnableBanding {
		for _, id := range existingBandedRangeIDs {
			reqs = append(reqs, &sheets.Request{
				DeleteBanding: &sheets.DeleteBandingRequest{
					BandedRangeId: id,
				},
			})
		}
	}

	frozenRows := spec.FrozenRows
	if frozenRows == 0 && spec.Data != nil && len(spec.Data.Headers) > 0 {
		frozenRows = 1
	}

	reqs = append(reqs, buildFreezeAndClearRequests(sheetID, frozenRows, spec.FrozenCols, totalRows)...)

	if hReq := buildHeaderFormatRequest(sheetID, frozenRows, theme); hReq != nil {
		reqs = append(reqs, hReq)
	}

	if bReq := buildBandingRequest(sheetID, frozenRows, totalRows, totalCols, theme); bReq != nil {
		reqs = append(reqs, bReq)
	}

	reqs = append(reqs, buildColumnDimensionRequests(sheetID, totalCols, spec, theme)...)

	return reqs
}

// buildRichLinkRequests creates UpdateCells batch requests to attach clickable URLs
// as rich-text links to cells with LinkURL without using spreadsheet formulas.
func buildRichLinkRequests(sheetID int64, table *Table) []*sheets.Request {
	if table == nil || len(table.Rows) == 0 {
		return nil
	}

	headerOffset := 0
	if len(table.Headers) > 0 {
		headerOffset = 1
	}

	var reqs []*sheets.Request
	for rIdx, row := range table.Rows {
		for cIdx, cell := range row {
			if cell.LinkURL == "" {
				continue
			}
			reqs = append(reqs, &sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Range: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    int64(rIdx + headerOffset),
						EndRowIndex:      int64(rIdx + headerOffset + 1),
						StartColumnIndex: int64(cIdx),
						EndColumnIndex:   int64(cIdx + 1),
					},
					Rows: []*sheets.RowData{
						{
							Values: []*sheets.CellData{
								{
									TextFormatRuns: []*sheets.TextFormatRun{
										{
											StartIndex: 0,
											Format: &sheets.TextFormat{
												Link: &sheets.Link{
													Uri: cell.LinkURL,
												},
											},
										},
									},
								},
							},
						},
					},
					Fields: fieldMaskTextFormatRuns,
				},
			})
		}
	}

	return reqs
}

func buildFreezeAndClearRequests(sheetID, frozenRows, frozenCols, totalRows int64) []*sheets.Request {
	reqs := []*sheets.Request{
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: sheetID,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount:    frozenRows,
						FrozenColumnCount: frozenCols,
					},
				},
				Fields: "gridProperties.frozenRowCount,gridProperties.frozenColumnCount",
			},
		},
	}

	if totalRows > frozenRows {
		reqs = append(reqs, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:       sheetID,
					StartRowIndex: frozenRows,
				},
				Cell:   &sheets.CellData{UserEnteredFormat: &sheets.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		})
	}

	return reqs
}

func buildHeaderFormatRequest(sheetID, frozenRows int64, theme *Theme) *sheets.Request {
	if frozenRows <= 0 {
		return nil
	}

	return &sheets.Request{
		RepeatCell: &sheets.RepeatCellRequest{
			Range: &sheets.GridRange{
				SheetId:       sheetID,
				StartRowIndex: 0,
				EndRowIndex:   frozenRows,
			},
			Cell: &sheets.CellData{
				UserEnteredFormat: &sheets.CellFormat{
					BackgroundColor: theme.HeaderBackground.ToSheetsColor(),
					TextFormat: &sheets.TextFormat{
						Bold:            theme.HeaderBold,
						ForegroundColor: theme.HeaderForeground.ToSheetsColor(),
					},
				},
			},
			Fields: "userEnteredFormat(backgroundColor,textFormat)",
		},
	}
}

func buildBandingRequest(sheetID, frozenRows, totalRows, totalCols int64, theme *Theme) *sheets.Request {
	if !theme.EnableBanding || totalRows <= frozenRows || totalCols <= 0 {
		return nil
	}

	return &sheets.Request{
		AddBanding: &sheets.AddBandingRequest{
			BandedRange: &sheets.BandedRange{
				Range: &sheets.GridRange{
					SheetId:          sheetID,
					StartRowIndex:    frozenRows,
					EndRowIndex:      totalRows,
					StartColumnIndex: 0,
					EndColumnIndex:   totalCols,
				},
				RowProperties: &sheets.BandingProperties{
					FirstBandColor:  theme.ZebraFirstBand.ToSheetsColor(),
					SecondBandColor: theme.ZebraSecondBand.ToSheetsColor(),
				},
			},
		},
	}
}

const (
	defaultEstWidth     int64 = 100
	minEstWidth         int64 = 60
	charPixelEstimate   int64 = 8
	cellPaddingEstimate int64 = 30
)

func buildColumnDimensionRequests(sheetID, totalCols int64, spec TabSpec, theme *Theme) []*sheets.Request {
	var reqs []*sheets.Request

	if theme.AutoFitColumns && totalCols > 0 {
		reqs = append(reqs, &sheets.Request{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: 0,
					EndIndex:   totalCols,
				},
			},
		})
	}

	widths := resolveFinalColumnWidths(totalCols, spec.Data, theme)
	for c := range totalCols {
		width, ok := widths[int(c)]
		if !ok || width <= 0 {
			continue
		}
		reqs = append(reqs, &sheets.Request{
			UpdateDimensionProperties: &sheets.UpdateDimensionPropertiesRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: c,
					EndIndex:   c + 1,
				},
				Properties: &sheets.DimensionProperties{
					PixelSize: width,
				},
				Fields: "pixelSize",
			},
		})
	}

	return reqs
}

func resolveFinalColumnWidths(totalCols int64, data *Table, theme *Theme) map[int]int64 {
	resolved := make(map[int]int64)
	if data == nil && theme == nil {
		return resolved
	}

	for colIdx := range int(totalCols) {
		w := computeColumnWidth(colIdx, data, theme)
		if w > 0 {
			resolved[colIdx] = w
		}
	}

	return resolved
}

func resolveColumnConstraint(colIdx int, data *Table, theme *Theme) ColumnConstraint {
	var constraint ColumnConstraint
	if data != nil && data.ColumnConstraints != nil {
		constraint = data.ColumnConstraints[colIdx]
	}
	if constraint.Width == 0 && data != nil && data.ColumnWidths != nil {
		constraint.Width = data.ColumnWidths[colIdx]
	}
	if constraint.Width == 0 && theme != nil && theme.ColumnWidths != nil {
		constraint.Width = theme.ColumnWidths[colIdx]
	}

	if constraint.MinWidth <= 0 && theme != nil {
		constraint.MinWidth = theme.MinColumnWidth
	}
	if constraint.MaxWidth <= 0 && theme != nil {
		constraint.MaxWidth = theme.MaxColumnWidth
	}

	return constraint
}

func computeColumnWidth(colIdx int, data *Table, theme *Theme) int64 {
	c := resolveColumnConstraint(colIdx, data, theme)

	if c.Width > 0 {
		return clampWidth(c.Width, c.MinWidth, c.MaxWidth)
	}

	if c.MinWidth > 0 || c.MaxWidth > 0 {
		estimated := estimateColumnContentWidth(data, colIdx)

		return clampWidth(estimated, c.MinWidth, c.MaxWidth)
	}

	return 0
}

func clampWidth(w, minW, maxW int64) int64 {
	if minW > 0 {
		w = max(w, minW)
	}
	if maxW > 0 {
		w = min(w, maxW)
	}

	return w
}

func estimateColumnContentWidth(data *Table, colIdx int) int64 {
	if data == nil {
		return defaultEstWidth
	}

	maxChars := 0
	if colIdx < len(data.Headers) {
		maxChars = len(data.Headers[colIdx])
	}

	for _, row := range data.Rows {
		if colIdx >= len(row) {
			continue
		}
		c := row[colIdx]
		s := fmt.Sprint(c.RawVal)
		if len(s) > maxChars {
			maxChars = len(s)
		}
	}

	estimated := int64(maxChars)*charPixelEstimate + cellPaddingEstimate

	return max(estimated, minEstWidth)
}

