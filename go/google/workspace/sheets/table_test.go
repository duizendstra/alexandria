package sheets

import (
	"testing"
	"time"
)

type Employee struct {
	ID        int       `sheets:"ID,width=80"`
	Name      string    `sheets:"Full Name,width=200"`
	Active    bool      `sheets:"Is Active"`
	JoinedAt  time.Time `sheets:"Join Date"`
	FormulaOp string    `sheets:"Formula Col,formula"`
	Internal  string    `sheets:"-"`
}

func TestFromStructs(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	items := []Employee{
		{
			ID:        101,
			Name:      "Alice Smith",
			Active:    true,
			JoinedAt:  now,
			FormulaOp: "=SUM(A2:B2)",
			Internal:  "secret-1",
		},
		{
			ID:        102,
			Name:      "=cmd|' /C calc'!A0", // Formula injection payload! Should be treated as RAW text.
			Active:    false,
			JoinedAt:  now,
			FormulaOp: "=AVERAGE(A3:B3)",
			Internal:  "secret-2",
		},
	}

	tbl, err := FromStructs(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHeaders := []string{"ID", "Full Name", "Is Active", "Join Date", "Formula Col"}
	if len(tbl.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(tbl.Headers))
	}
	for i, h := range expectedHeaders {
		if tbl.Headers[i] != h {
			t.Errorf("header %d: got %q, want %q", i, tbl.Headers[i], h)
		}
	}

	if tbl.ColumnWidths[0] != 80 {
		t.Errorf("expected col 0 width 80, got %d", tbl.ColumnWidths[0])
	}
	if tbl.ColumnWidths[1] != 200 {
		t.Errorf("expected col 1 width 200, got %d", tbl.ColumnWidths[1])
	}

	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}

	// Verify Row 0.
	row0 := tbl.Rows[0]
	if row0[0].RawVal != int64(101) || row0[0].IsFormula {
		t.Errorf("row 0 col 0: expected 101 raw, got %+v", row0[0])
	}
	if row0[1].RawVal != "Alice Smith" || row0[1].IsFormula {
		t.Errorf("row 0 col 1: expected Alice Smith raw, got %+v", row0[1])
	}
	if row0[4].RawVal != "=SUM(A2:B2)" || !row0[4].IsFormula {
		t.Errorf("row 0 col 4: expected formula flag true, got %+v", row0[4])
	}

	// Verify Row 1 Formula Injection Immunity.
	row1 := tbl.Rows[1]
	if row1[1].RawVal != "=cmd|' /C calc'!A0" || row1[1].IsFormula {
		t.Errorf("row 1 col 1: untrusted input must NOT be marked as formula, got %+v", row1[1])
	}
}

func TestFluentTableBuilder(t *testing.T) {
	tbl := NewTable("Name", "Score", "Link")
	tbl.AddRowValues("Alice", 95, Hyperlink("https://example.com/alice", "Profile"))
	tbl.AddRowValues("=Bob", 88, Hyperlink("https://example.com/bob", "Profile"))
	tbl.SetColumnWidth(0, 150)

	if tbl.RowCount() != 2 {
		t.Errorf("expected 2 rows, got %d", tbl.RowCount())
	}
	if tbl.ColCount() != 3 {
		t.Errorf("expected 3 cols, got %d", tbl.ColCount())
	}

	// "=Bob" must not be a formula.
	if tbl.Rows[1][0].IsFormula {
		t.Errorf("expected '=Bob' to be raw text, got formula")
	}

	// Hyperlink must be rich text (IsFormula: false, LinkURL set).
	linkCell := tbl.Rows[0][2]
	if linkCell.IsFormula {
		t.Errorf("expected hyperlink to be rich text with IsFormula=false")
	}
	if linkCell.RawVal != "Profile" {
		t.Errorf("got hyperlink label %q, want %q", linkCell.RawVal, "Profile")
	}
	if linkCell.LinkURL != "https://example.com/alice" {
		t.Errorf("got link URL %q, want %q", linkCell.LinkURL, "https://example.com/alice")
	}
}

func TestHyperlink_FallbackLabel(t *testing.T) {
	link := Hyperlink("https://example.com/item", "")
	if link.RawVal != "https://example.com/item" {
		t.Errorf("got %q, want %q", link.RawVal, "https://example.com/item")
	}
	if link.LinkURL != "https://example.com/item" {
		t.Errorf("got LinkURL %q, want %q", link.LinkURL, "https://example.com/item")
	}
	if link.IsFormula {
		t.Errorf("expected IsFormula=false for rich-text hyperlink")
	}
}

func TestTable_SetColumnWidthByName(t *testing.T) {
	tbl := NewTable("First Name", "Last Name", "Email Address")
	tbl.SetColumnWidthByName("Last Name", 160)
	tbl.SetColumnWidthByName("email address", 250) // test case-insensitivity.
	tbl.SetColumnWidthByName("NonExistent", 300)

	if tbl.ColumnWidths[1] != 160 {
		t.Errorf("expected col 1 width 160, got %d", tbl.ColumnWidths[1])
	}
	if tbl.ColumnWidths[2] != 250 {
		t.Errorf("expected col 2 width 250, got %d", tbl.ColumnWidths[2])
	}
	if _, ok := tbl.ColumnWidths[3]; ok {
		t.Errorf("expected non-existent header not to be added to widths")
	}
}

type ConstrainedItem struct {
	Title string `sheets:"Title,minWidth=100,maxWidth=400"`
	Code  string `sheets:"Code,width=80"`
	Notes string `sheets:"Notes,max=300"`
}

func TestFromStructs_ColumnConstraints(t *testing.T) {
	items := []ConstrainedItem{
		{Title: "Large Migration Document", Code: "MIG-01", Notes: "Some long note here"},
	}

	tbl, err := FromStructs(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Col 0: Title (min 100, max 400).
	c0 := tbl.ColumnConstraints[0]
	if c0.MinWidth != 100 || c0.MaxWidth != 400 {
		t.Errorf("col 0 constraints: got %+v, want min=100 max=400", c0)
	}

	// Col 1: Code (fixed width 80).
	c1 := tbl.ColumnConstraints[1]
	if c1.Width != 80 {
		t.Errorf("col 1 constraints: got %+v, want width=80", c1)
	}

	// Col 2: Notes (max 300).
	c2 := tbl.ColumnConstraints[2]
	if c2.MaxWidth != 300 {
		t.Errorf("col 2 constraints: got %+v, want max=300", c2)
	}
}

func TestTable_SetColumnBoundsByName(t *testing.T) {
	tbl := NewTable("Topic", "Summary")
	tbl.SetColumnBoundsByName("Topic", 120, 300)
	tbl.SetColumnMaxWidthByName("Summary", 500)
	tbl.SetColumnMinWidthByName("Summary", 150)

	c0 := tbl.ColumnConstraints[0]
	if c0.MinWidth != 120 || c0.MaxWidth != 300 {
		t.Errorf("col 0: got %+v, want min=120 max=300", c0)
	}

	c1 := tbl.ColumnConstraints[1]
	if c1.MinWidth != 150 || c1.MaxWidth != 500 {
		t.Errorf("col 1: got %+v, want min=150 max=500", c1)
	}
}


