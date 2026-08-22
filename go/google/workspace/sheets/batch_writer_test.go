package sheets

import (
	"fmt"
	"reflect"
	"testing"
)

const (
	testOptRaw = "RAW"

	testHeaderKey   = "Key"
	testHeaderValue = "Value"
	testFormulaSum  = "=SUM(1,2)"
	testRangeB2     = "'T'!B2:B2"
)

func TestColIndexToA1(t *testing.T) {
	tests := []struct {
		colIdx int
		want   string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
	}

	for _, tt := range tests {
		got := ColIndexToA1(tt.colIdx)
		if got != tt.want {
			t.Errorf("ColIndexToA1(%d) = %q, want %q", tt.colIdx, got, tt.want)
		}
	}
}

func TestEscapeSheetTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{DefaultSheetTitle, "'Sheet1'"},
		{"My 'Data' Tab", "'My ''Data'' Tab'"},
		{"Users 2026", "'Users 2026'"},
	}

	for _, tt := range tests {
		got := EscapeSheetTitle(tt.title)
		if got != tt.want {
			t.Errorf("EscapeSheetTitle(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestPrepareValueUpdates_FastPathAllRaw(t *testing.T) {
	tbl := NewTable("Name", "Age", "Role")
	tbl.AddRowValues("Alice", 30, "Engineer")
	tbl.AddRowValues("Bob", 40, "Manager")

	batches := prepareValueUpdates("Staff", tbl)
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch for all-raw table, got %d", len(batches))
	}

	b := batches[0]
	if b.Range != "'Staff'!A1" {
		t.Errorf("expected range 'Staff'!A1, got %q", b.Range)
	}
	if b.ValueInputOption != testOptRaw {
		t.Errorf("expected ValueInputOption %s, got %q", testOptRaw, b.ValueInputOption)
	}
	if len(b.Values) != 3 { // 1 header + 2 data rows.
		t.Fatalf("expected 3 rows in values, got %d", len(b.Values))
	}

	expectedHeader := []any{"Name", "Age", "Role"}
	if !reflect.DeepEqual(b.Values[0], expectedHeader) {
		t.Errorf("header mismatch: got %v, want %v", b.Values[0], expectedHeader)
	}
}

func TestPrepareValueUpdates_MixedFormulas(t *testing.T) {
	// Columns:
	// 0: First Name (RAW)
	// 1: Last Name (RAW)
	// 2: Full Name Formula (USER_ENTERED)
	// 3: Status (RAW)
	tbl := NewTable("First", "Last", "Full Name", "Status")
	tbl.AddRow(Text("Alice"), Text("Smith"), Formula(`=CONCATENATE(A2, " ", B2)`), Text("Active"))
	tbl.AddRow(Text("Bob"), Text("Jones"), Formula(`=CONCATENATE(A3, " ", B3)`), Text("Pending"))

	batches := prepareValueUpdates("Users", tbl)

	// Should produce 4 batches:
	// 1. Header (A1:D1) -> RAW
	// 2. Col 0-1 (A2:B3) -> RAW (First, Last)
	// 3. Col 2 (C2:C3) -> USER_ENTERED (Full Name formula)
	// 4. Col 3 (D2:D3) -> RAW (Status).
	if len(batches) != 4 {
		t.Fatalf("expected 4 batches for mixed table, got %d", len(batches))
	}

	// 1. Header.
	if batches[0].Range != "'Users'!A1:D1" || batches[0].ValueInputOption != testOptRaw {
		t.Errorf("batch 0: expected 'Users'!A1:D1 %s, got %s %s", testOptRaw, batches[0].Range, batches[0].ValueInputOption)
	}

	// 2. A2:B3 RAW.
	if batches[1].Range != "'Users'!A2:B3" || batches[1].ValueInputOption != testOptRaw {
		t.Errorf("batch 1: expected 'Users'!A2:B3 %s, got %s %s", testOptRaw, batches[1].Range, batches[1].ValueInputOption)
	}

	// 3. C2:C3 USER_ENTERED.
	if batches[2].Range != "'Users'!C2:C3" || batches[2].ValueInputOption != "USER_ENTERED" {
		t.Errorf("batch 2: expected 'Users'!C2:C3 USER_ENTERED, got %s %s", batches[2].Range, batches[2].ValueInputOption)
	}

	// 4. D2:D3 RAW.
	if batches[3].Range != "'Users'!D2:D3" || batches[3].ValueInputOption != testOptRaw {
		t.Errorf("batch 3: expected 'Users'!D2:D3 %s, got %s %s", testOptRaw, batches[3].Range, batches[3].ValueInputOption)
	}
}

// TestPrepareValueUpdates_HomogeneousColumnsGolden pins the exact requests emitted for
// columns that are purely text or purely formula, including a short row whose missing
// cell is padded with "". These requests must not change when mixed columns are split.
func TestPrepareValueUpdates_HomogeneousColumnsGolden(t *testing.T) {
	tbl := NewTable("Item", "Total", "Note")
	tbl.AddRow(Text("a"), Formula(testFormulaSum), Text("=looks-like-a-formula"))
	tbl.AddRow(Text("b"), Formula("=SUM(3,4)"))
	tbl.AddRow(Text("c"), Formula("=SUM(5,6)"), Text("+plus"))

	want := []valueUpdateBatch{
		{Range: "'T'!A1:C1", ValueInputOption: testOptRaw, Values: [][]any{{"Item", "Total", "Note"}}},
		{Range: "'T'!A2:A4", ValueInputOption: testOptRaw, Values: [][]any{{"a"}, {"b"}, {"c"}}},
		{Range: "'T'!B2:B4", ValueInputOption: InputOptionUserEntered, Values: [][]any{{testFormulaSum}, {"=SUM(3,4)"}, {"=SUM(5,6)"}}},
		{Range: "'T'!C2:C4", ValueInputOption: testOptRaw, Values: [][]any{{"=looks-like-a-formula"}, {""}, {"+plus"}}},
	}

	got := prepareValueUpdates("T", tbl)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("homogeneous columns: got\n%#v\nwant\n%#v", got, want)
	}
}

// TestPrepareValueUpdates_MixedColumnPerCellInput covers a column holding both formula
// and text cells: only the formula cells may be sent USER_ENTERED; every text cell —
// including ones that start with a formula trigger — must land in a RAW request so
// Sheets stores it literally (#244).
func TestPrepareValueUpdates_MixedColumnPerCellInput(t *testing.T) {
	tests := []struct {
		name  string
		build func() *Table
		want  []valueUpdateBatch
	}{
		{
			name: "alternating runs",
			build: func() *Table {
				tbl := NewTable(testHeaderKey, testHeaderValue)
				tbl.AddRow(Text("a"), Formula(testFormulaSum))
				tbl.AddRow(Text("b"), Text("=not-a-formula"))
				tbl.AddRow(Text("c"), Formula("=SUM(3,4)"))
				tbl.AddRow(Text("d"), Text("+tel"))
				tbl.AddRow(Text("e"), Text("-neg"))

				return tbl
			},
			want: []valueUpdateBatch{
				{Range: "'T'!A1:B1", ValueInputOption: testOptRaw, Values: [][]any{{testHeaderKey, testHeaderValue}}},
				{Range: "'T'!A2:A6", ValueInputOption: testOptRaw, Values: [][]any{{"a"}, {"b"}, {"c"}, {"d"}, {"e"}}},
				{Range: testRangeB2, ValueInputOption: InputOptionUserEntered, Values: [][]any{{testFormulaSum}}},
				{Range: "'T'!B3:B3", ValueInputOption: testOptRaw, Values: [][]any{{"=not-a-formula"}}},
				{Range: "'T'!B4:B4", ValueInputOption: InputOptionUserEntered, Values: [][]any{{"=SUM(3,4)"}}},
				{Range: "'T'!B5:B6", ValueInputOption: testOptRaw, Values: [][]any{{"+tel"}, {"-neg"}}},
			},
		},
		{
			name: "short row pads the mixed column with an empty RAW cell",
			build: func() *Table {
				tbl := NewTable(testHeaderKey, testHeaderValue)
				tbl.AddRow(Text("a"), Formula(testFormulaSum))
				tbl.AddRow(Text("b"))
				tbl.AddRow(Text("c"), Text("=x"))

				return tbl
			},
			want: []valueUpdateBatch{
				{Range: "'T'!A1:B1", ValueInputOption: testOptRaw, Values: [][]any{{testHeaderKey, testHeaderValue}}},
				{Range: "'T'!A2:A4", ValueInputOption: testOptRaw, Values: [][]any{{"a"}, {"b"}, {"c"}}},
				{Range: testRangeB2, ValueInputOption: InputOptionUserEntered, Values: [][]any{{testFormulaSum}}},
				{Range: "'T'!B3:B4", ValueInputOption: testOptRaw, Values: [][]any{{""}, {"=x"}}},
			},
		},
		{
			name: "mixed column between homogeneous neighbours keeps their grouping",
			build: func() *Table {
				tbl := NewTable("A", "B", "C", "D")
				tbl.AddRow(Text("a1"), Text("=b1"), Formula("=1"), Formula("=2"))
				tbl.AddRow(Text("a2"), Formula("=3"), Formula("=4"), Formula("=5"))

				return tbl
			},
			want: []valueUpdateBatch{
				{Range: "'T'!A1:D1", ValueInputOption: testOptRaw, Values: [][]any{{"A", "B", "C", "D"}}},
				{Range: "'T'!A2:A3", ValueInputOption: testOptRaw, Values: [][]any{{"a1"}, {"a2"}}},
				{Range: testRangeB2, ValueInputOption: testOptRaw, Values: [][]any{{"=b1"}}},
				{Range: "'T'!B3:B3", ValueInputOption: InputOptionUserEntered, Values: [][]any{{"=3"}}},
				{Range: "'T'!C2:D3", ValueInputOption: InputOptionUserEntered, Values: [][]any{{"=1", "=2"}, {"=4", "=5"}}},
			},
		},
		{
			name: "headerless table starts the runs on row 1",
			build: func() *Table {
				tbl := NewTable()
				tbl.AddRow(Text("=t"), Formula("=1"))
				tbl.AddRow(Formula("=2"), Text("@u"))

				return tbl
			},
			want: []valueUpdateBatch{
				{Range: "'T'!A1:A1", ValueInputOption: testOptRaw, Values: [][]any{{"=t"}}},
				{Range: "'T'!A2:A2", ValueInputOption: InputOptionUserEntered, Values: [][]any{{"=2"}}},
				{Range: "'T'!B1:B1", ValueInputOption: InputOptionUserEntered, Values: [][]any{{"=1"}}},
				{Range: testRangeB2, ValueInputOption: testOptRaw, Values: [][]any{{"@u"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareValueUpdates("T", tt.build())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got\n%#v\nwant\n%#v", got, tt.want)
			}
			assertOnlyFormulasUserEntered(t, tt.build(), got)
		})
	}
}

// assertOnlyFormulasUserEntered checks the invariant behind #244: a USER_ENTERED request
// may carry nothing but cells the table marked IsFormula.
func assertOnlyFormulasUserEntered(t *testing.T, tbl *Table, batches []valueUpdateBatch) {
	t.Helper()
	formulas := map[string]bool{}
	for _, r := range tbl.Rows {
		for _, c := range r {
			if c.IsFormula {
				formulas[fmt.Sprint(c.RawVal)] = true
			}
		}
	}
	for _, b := range batches {
		if b.ValueInputOption != InputOptionUserEntered {
			continue
		}
		for _, row := range b.Values {
			for _, v := range row {
				if !formulas[fmt.Sprint(v)] {
					t.Errorf("range %s sent %q as USER_ENTERED but it is not a formula cell", b.Range, v)
				}
			}
		}
	}
}
