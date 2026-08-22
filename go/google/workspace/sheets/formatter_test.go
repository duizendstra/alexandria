package sheets

import (
	"testing"
)

func TestBuildFormatRequests_ThemeCorporateNavy(t *testing.T) {
	tbl := NewTable("Col1", "Col2", "Col3")
	tbl.AddRowValues("a", "b", "c")
	tbl.SetColumnWidth(0, 180)

	spec := TabSpec{
		Title:      "Overview",
		FrozenRows: 1,
		FrozenCols: 0,
		Theme:      ThemeCorporateNavy(),
		Data:       tbl,
	}

	reqs := buildFormatRequests(12345, spec, 2, 3)
	if len(reqs) == 0 {
		t.Fatalf("expected format requests to be generated")
	}

	// 1. UpdateSheetProperties (freeze).
	if reqs[0].UpdateSheetProperties == nil {
		t.Errorf("req 0: expected UpdateSheetProperties")
	} else {
		gp := reqs[0].UpdateSheetProperties.Properties.GridProperties
		if gp.FrozenRowCount != 1 {
			t.Errorf("expected frozen row count 1, got %d", gp.FrozenRowCount)
		}
	}

	// 2. RepeatCell (clear format below row 1).
	if reqs[1].RepeatCell == nil || reqs[1].RepeatCell.Range.StartRowIndex != 1 {
		t.Errorf("req 1: expected RepeatCell from row index 1")
	}

	// 3. RepeatCell (header formatting).
	if reqs[2].RepeatCell == nil || reqs[2].RepeatCell.Range.EndRowIndex != 1 {
		t.Errorf("req 2: expected RepeatCell header formatting")
	} else {
		tf := reqs[2].RepeatCell.Cell.UserEnteredFormat.TextFormat
		if !tf.Bold {
			t.Errorf("expected header text bold")
		}
	}

	// 4. AddBanding.
	if reqs[3].AddBanding == nil {
		t.Errorf("req 3: expected AddBanding")
	}

	// 5. AutoResizeDimensions.
	if reqs[4].AutoResizeDimensions == nil {
		t.Errorf("req 4: expected AutoResizeDimensions")
	}

	// 6. UpdateDimensionProperties (column width 180 on col 0).
	if reqs[5].UpdateDimensionProperties == nil {
		t.Errorf("req 5: expected UpdateDimensionProperties for col 0")
	} else {
		dim := reqs[5].UpdateDimensionProperties
		if dim.Properties.PixelSize != 180 || dim.Range.StartIndex != 0 {
			t.Errorf("got width %d on col %d, want 180 on col 0", dim.Properties.PixelSize, dim.Range.StartIndex)
		}
	}
}
