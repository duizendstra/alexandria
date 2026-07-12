package datadiff_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

func TestDiffColumns_IdenticalSchemas(t *testing.T) {
	cols := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "name", DataType: "STRING"},
		{Name: "amount", DataType: "FLOAT64"},
	}

	result := datadiff.DiffColumns(cols, cols)
	if !result.Match {
		t.Error("identical schemas should match")
	}
	if len(result.LeftOnly) != 0 || len(result.RightOnly) != 0 || len(result.TypeDiffs) != 0 {
		t.Errorf("expected no diffs, got left=%d right=%d types=%d",
			len(result.LeftOnly), len(result.RightOnly), len(result.TypeDiffs))
	}
}

func TestDiffColumns_LeftOnlyColumn(t *testing.T) {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "legacy_field", DataType: "STRING"},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
	}

	result := datadiff.DiffColumns(left, right)
	if result.Match {
		t.Error("should not match with left-only column")
	}
	if len(result.LeftOnly) != 1 || result.LeftOnly[0].Name != "legacy_field" {
		t.Errorf("expected legacy_field as left-only, got %v", result.LeftOnly)
	}
}

func TestDiffColumns_RightOnlyColumn(t *testing.T) {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "new_field", DataType: "STRING"},
	}

	result := datadiff.DiffColumns(left, right)
	if result.Match {
		t.Error("should not match with right-only column")
	}
	if len(result.RightOnly) != 1 || result.RightOnly[0].Name != "new_field" {
		t.Errorf("expected new_field as right-only, got %v", result.RightOnly)
	}
}

func TestDiffColumns_TypeMismatch(t *testing.T) {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "amount", DataType: "FLOAT64"},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "amount", DataType: "NUMERIC"},
	}

	result := datadiff.DiffColumns(left, right)
	if result.Match {
		t.Error("should not match with type difference")
	}
	if len(result.TypeDiffs) != 1 {
		t.Fatalf("expected 1 type diff, got %d", len(result.TypeDiffs))
	}
	td := result.TypeDiffs[0]
	if td.Name != "amount" || td.LeftType != "FLOAT64" || td.RightType != "NUMERIC" {
		t.Errorf("unexpected type diff: %+v", td)
	}
}

func TestDiffColumns_NestedRecord(t *testing.T) {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "address", DataType: "RECORD", Children: []datadiff.Column{
			{Name: "city", DataType: "STRING"},
			{Name: "zip", DataType: "STRING"},
		}},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "address", DataType: "RECORD", Children: []datadiff.Column{
			{Name: "city", DataType: "STRING"},
		}},
	}

	result := datadiff.DiffColumns(left, right)
	if result.Match {
		t.Error("should not match when nested field is missing")
	}
	if len(result.LeftOnly) != 1 || result.LeftOnly[0].Name != "address.zip" {
		t.Errorf("expected address.zip as left-only, got %v", result.LeftOnly)
	}
}

func TestDiffColumns_EmptySchemas(t *testing.T) {
	result := datadiff.DiffColumns(nil, nil)
	if !result.Match {
		t.Error("two empty schemas should match")
	}
}

func TestDiffColumns_MultipleDiffs(t *testing.T) {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "old_col", DataType: "STRING"},
		{Name: "price", DataType: "FLOAT64"},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "new_col", DataType: "STRING"},
		{Name: "price", DataType: "NUMERIC"},
	}

	result := datadiff.DiffColumns(left, right)
	if result.Match {
		t.Error("should not match")
	}
	if len(result.LeftOnly) != 1 {
		t.Errorf("expected 1 left-only, got %d", len(result.LeftOnly))
	}
	if len(result.RightOnly) != 1 {
		t.Errorf("expected 1 right-only, got %d", len(result.RightOnly))
	}
	if len(result.TypeDiffs) != 1 {
		t.Errorf("expected 1 type diff, got %d", len(result.TypeDiffs))
	}
}

func TestDiffStats_IdenticalStats(t *testing.T) {
	stats := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1000, Avg: 50, Min: 1, Max: 100, CountDistinct: 80, NullCount: 0},
		{Column: "quantity", Sum: 500, Avg: 25, Min: 1, Max: 50, CountDistinct: 40, NullCount: 5},
	}

	result := datadiff.DiffStats(stats, stats, 0)
	if !result.Match {
		t.Error("identical stats should match")
	}
	if len(result.Diffs) != 0 {
		t.Errorf("expected no diffs, got %d", len(result.Diffs))
	}
}

func TestDiffStats_SumDifference(t *testing.T) {
	left := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1000, Avg: 50, Min: 1, Max: 100, CountDistinct: 80, NullCount: 0},
	}
	right := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1050, Avg: 50, Min: 1, Max: 100, CountDistinct: 80, NullCount: 0},
	}

	result := datadiff.DiffStats(left, right, 0)
	if result.Match {
		t.Error("should not match with sum difference")
	}

	found := false
	for _, d := range result.Diffs {
		if d.Column == "amount" && d.Field == "Sum" {
			found = true
			if d.Delta != 50 {
				t.Errorf("expected delta=50, got %f", d.Delta)
			}
		}
	}
	if !found {
		t.Error("expected Sum diff for amount column")
	}
}

func TestDiffStats_WithTolerance(t *testing.T) {
	left := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1000, Avg: 50, Min: 1, Max: 100, CountDistinct: 80, NullCount: 0},
	}
	right := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1005, Avg: 50, Min: 1, Max: 100, CountDistinct: 80, NullCount: 0},
	}

	// 0.5% diff with 1% tolerance — should pass.
	result := datadiff.DiffStats(left, right, 0.01)
	if !result.Match {
		t.Error("0.5% diff should match with 1% tolerance")
	}

	// 0.5% diff with 0.1% tolerance — should fail.
	result = datadiff.DiffStats(left, right, 0.001)
	if result.Match {
		t.Error("0.5% diff should not match with 0.1% tolerance")
	}
}

func TestDiffStats_NullCountDifference(t *testing.T) {
	left := []datadiff.ColumnStat{
		{Column: "price", Sum: 500, Avg: 25, Min: 1, Max: 50, CountDistinct: 20, NullCount: 3},
	}
	right := []datadiff.ColumnStat{
		{Column: "price", Sum: 500, Avg: 25, Min: 1, Max: 50, CountDistinct: 20, NullCount: 5},
	}

	result := datadiff.DiffStats(left, right, 0)
	if result.Match {
		t.Error("should not match with null count difference")
	}

	found := false
	for _, d := range result.Diffs {
		if d.Column == "price" && d.Field == "NullCount" {
			found = true
		}
	}
	if !found {
		t.Error("expected NullCount diff for price column")
	}
}

func TestDiffStats_MissingColumn(t *testing.T) {
	left := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1000},
		{Column: "quantity", Sum: 500},
	}
	right := []datadiff.ColumnStat{
		{Column: "amount", Sum: 1000},
	}

	result := datadiff.DiffStats(left, right, 0)
	if !result.Match {
		t.Error("should match — missing columns are skipped (schema layer handles this)")
	}
}
