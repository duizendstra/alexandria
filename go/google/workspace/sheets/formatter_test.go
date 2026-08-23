package sheets

import (
	"testing"

	googlesheets "google.golang.org/api/sheets/v4"
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

	reqs := buildFormatRequests(12345, nil, spec, 2, 3)
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

func TestBuildFormatRequests_DeleteExistingBanding(t *testing.T) {
	spec := TabSpec{
		Title: "BandedTab",
		Theme: ThemeModernSlate(),
	}

	reqs := buildFormatRequests(999, []int64{101, 102}, spec, 5, 2)
	if len(reqs) < 2 {
		t.Fatalf("expected at least 2 requests for deleting bandings")
	}

	if reqs[0].DeleteBanding == nil || reqs[0].DeleteBanding.BandedRangeId != 101 {
		t.Errorf("expected DeleteBanding for id 101, got %+v", reqs[0].DeleteBanding)
	}
	if reqs[1].DeleteBanding == nil || reqs[1].DeleteBanding.BandedRangeId != 102 {
		t.Errorf("expected DeleteBanding for id 102, got %+v", reqs[1].DeleteBanding)
	}
}

func TestBuildFormatRequests_ColumnBoundsAndMax(t *testing.T) {
	tbl := NewTable("LongColumn", "ShortColumn", "FixedBounded")
	// Col 0 has very long text (over 100 chars).
	tbl.AddRowValues("https://example.com/very/long/url/path/with/lots/of/parameters?query=value&another=something_extremely_verbose", "hi", "normal")
	tbl.SetColumnBounds(0, 50, 220)  // Col 0: bounded to max 220px.
	tbl.SetColumnBounds(1, 140, 500) // Col 1: bounded to min 140px.
	tbl.SetColumnWidth(2, 400)       // Col 2: fixed 400.
	tbl.SetColumnBounds(2, 0, 250)   // Col 2: max 250 (should clamp the 400 down to 250).

	spec := TabSpec{
		Title: "BoundedTab",
		Data:  tbl,
	}

	reqs := buildFormatRequests(111, nil, spec, 2, 3)

	var colUpdates []*googlesheets.UpdateDimensionPropertiesRequest
	for _, r := range reqs {
		if r.UpdateDimensionProperties != nil {
			colUpdates = append(colUpdates, r.UpdateDimensionProperties)
		}
	}

	if len(colUpdates) != 3 {
		t.Fatalf("expected 3 column updates, got %d", len(colUpdates))
	}

	// Col 0: clamped to max 220.
	if colUpdates[0].Properties.PixelSize != 220 {
		t.Errorf("col 0: got %d, want 220", colUpdates[0].Properties.PixelSize)
	}

	// Col 1: clamped to min 140.
	if colUpdates[1].Properties.PixelSize != 140 {
		t.Errorf("col 1: got %d, want 140", colUpdates[1].Properties.PixelSize)
	}

	// Col 2: fixed 400 clamped to max 250.
	if colUpdates[2].Properties.PixelSize != 250 {
		t.Errorf("col 2: got %d, want 250", colUpdates[2].Properties.PixelSize)
	}
}

func TestBuildFormatRequests_SkipFormatting(t *testing.T) {
	tbl := NewTable("Col1", "Col2")
	tbl.AddRowValues("a", "b")

	spec := TabSpec{
		Title:          "RawOnlyTab",
		SkipFormatting: true,
		Data:           tbl,
	}

	reqs := buildFormatRequests(123, []int64{101, 102}, spec, 2, 2)
	if len(reqs) != 0 {
		t.Fatalf("expected 0 format requests with SkipFormatting=true, got %d", len(reqs))
	}
}

func TestBuildFormatRequests_ThemeCorporateNavyPlain(t *testing.T) {
	tbl := NewTable("Col1", "Col2")
	tbl.AddRowValues("a", "b")

	theme := ThemeCorporateNavyPlain()
	if theme.EnableBanding {
		t.Errorf("expected EnableBanding to be false for Plain theme")
	}

	spec := TabSpec{
		Title: "PlainNavyTab",
		Theme: theme,
		Data:  tbl,
	}

	reqs := buildFormatRequests(123, []int64{101, 102}, spec, 2, 2)

	for _, r := range reqs {
		if r.AddBanding != nil {
			t.Errorf("expected zero AddBanding requests, but found one: %+v", r.AddBanding)
		}
		if r.DeleteBanding != nil {
			t.Errorf("expected zero DeleteBanding requests when banding is disabled, but found: %+v", r.DeleteBanding)
		}
	}
}

func TestBuildRichLinkRequests(t *testing.T) {
	tbl := NewTable("Name", "Profile")
	tbl.AddRow(Text("Alice"), Hyperlink("https://example.com/alice", "Alice Profile"))
	tbl.AddRow(Text("Bob"), Text("No Link"))

	reqs := buildRichLinkRequests(456, tbl)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 rich link request, got %d", len(reqs))
	}

	up := reqs[0].UpdateCells
	if up == nil {
		t.Fatalf("expected UpdateCells request")
	}
	if up.Range.StartRowIndex != 1 || up.Range.EndRowIndex != 2 {
		t.Errorf("got row range [%d, %d), want [1, 2)", up.Range.StartRowIndex, up.Range.EndRowIndex)
	}
	if up.Range.StartColumnIndex != 1 || up.Range.EndColumnIndex != 2 {
		t.Errorf("got col range [%d, %d), want [1, 2)", up.Range.StartColumnIndex, up.Range.EndColumnIndex)
	}
	if len(up.Rows) != 1 || len(up.Rows[0].Values) != 1 {
		t.Fatalf("invalid row/value count in UpdateCells")
	}

	tfr := up.Rows[0].Values[0].TextFormatRuns
	if len(tfr) != 1 {
		t.Fatalf("expected 1 text format run, got %d", len(tfr))
	}
	if tfr[0].Format.Link == nil || tfr[0].Format.Link.Uri != "https://example.com/alice" {
		t.Errorf("expected link uri https://example.com/alice, got %+v", tfr[0].Format.Link)
	}
	if up.Fields != fieldMaskTextFormatRuns {
		t.Errorf("expected fields %q, got %q", fieldMaskTextFormatRuns, up.Fields)
	}
}
