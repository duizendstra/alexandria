package sheets

import (
	"maps"

	"google.golang.org/api/sheets/v4"
)

// buildFormatRequests creates the Google Sheets API batch update requests
// to apply theme styling, frozen panes, zebra banding, and column dimensions.
func buildFormatRequests(sheetID int64, existingBandedRangeIDs []int64, spec TabSpec, totalRows, totalCols int64) []*sheets.Request {
	var reqs []*sheets.Request

	// 0. Delete any prior banding ranges on this tab to avoid conflict on idempotency.
	for _, id := range existingBandedRangeIDs {
		reqs = append(reqs, &sheets.Request{
			DeleteBanding: &sheets.DeleteBandingRequest{
				BandedRangeId: id,
			},
		})
	}

	theme := spec.Theme
	if theme == nil {
		theme = ThemeCorporateNavy()
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
