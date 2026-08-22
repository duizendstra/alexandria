package sheets

import (
	"maps"

	"google.golang.org/api/sheets/v4"
)

// buildFormatRequests creates the Google Sheets API batch update requests
// to apply theme styling, frozen panes, zebra banding, and column dimensions.
func buildFormatRequests(sheetID int64, spec TabSpec, totalRows, totalCols int64) []*sheets.Request {
	var reqs []*sheets.Request

	theme := spec.Theme
	if theme == nil {
		theme = ThemeCorporateNavy()
	}

	frozenRows := spec.FrozenRows
	if frozenRows == 0 && spec.Data != nil && len(spec.Data.Headers) > 0 {
		frozenRows = 1
	}

	// 1. Frozen rows and columns.
	reqs = append(reqs, &sheets.Request{
		UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			Properties: &sheets.SheetProperties{
				SheetId: sheetID,
				GridProperties: &sheets.GridProperties{
					FrozenRowCount:    frozenRows,
					FrozenColumnCount: spec.FrozenCols,
				},
			},
			Fields: "gridProperties.frozenRowCount,gridProperties.frozenColumnCount",
		},
	})

	// 2. Clear cell formatting below header rows.
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

	// 3. Header row formatting.
	if frozenRows > 0 {
		reqs = append(reqs, &sheets.Request{
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
		})
	}

	// 4. Alternating row banding.
	if theme.EnableBanding && totalRows > frozenRows && totalCols > 0 {
		reqs = append(reqs, &sheets.Request{
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
		})
	}

	// 5. Auto-fit column widths.
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

	// 6. Custom column width overrides.
	widths := collectColumnWidths(theme, spec.Data)
	for colIdx, width := range widths {
		if width <= 0 {
			continue
		}
		c := int64(colIdx)
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

func collectColumnWidths(theme *Theme, data *Table) map[int]int64 {
	widths := make(map[int]int64)
	if theme != nil {
		maps.Copy(widths, theme.ColumnWidths)
	}
	if data != nil {
		maps.Copy(widths, data.ColumnWidths)
	}

	return widths
}
