package sheets

import (
	"reflect"
	"testing"
)

const (
	testOptRaw = "RAW"
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
