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

	// Hyperlink must be a formula.
	linkCell := tbl.Rows[0][2]
	if !linkCell.IsFormula {
		t.Errorf("expected hyperlink to be marked as formula")
	}
	wantFormula := `=HYPERLINK("https://example.com/alice";"Profile")`
	if linkCell.RawVal != wantFormula {
		t.Errorf("got hyperlink %q, want %q", linkCell.RawVal, wantFormula)
	}
}

func TestHyperlinkQuoteEscaping(t *testing.T) {
	link := Hyperlink("https://example.com/item?q=\"search\"", "Look \"Here\"")
	want := `=HYPERLINK("https://example.com/item?q=""search""";"Look ""Here""")`
	if link.RawVal != want {
		t.Errorf("got %q, want %q", link.RawVal, want)
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

