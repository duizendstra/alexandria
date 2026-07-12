package datadiff

import "fmt"

// ComparisonSpec describes what was compared.
type ComparisonSpec struct {
	LeftName  string
	RightName string
	Key       string
	Filter    string
}

// Result is the outcome of a full comparison across all layers.
type Result struct {
	Spec    ComparisonSpec
	Schema  SchemaResult
	Volume  VolumeResult
	Content ContentResult
	Stats   StatsResult
}

// Pass returns true if all deterministic layers match.
func (r *Result) Pass() bool {
	return r.Schema.Match && r.Volume.Match && r.Content.Match
}

// SchemaResult compares table structure.
type SchemaResult struct {
	Match     bool
	LeftOnly  []Column // columns only in left.
	RightOnly []Column // columns only in right.
	TypeDiffs []ColumnTypeDiff
}

// ColumnTypeDiff records a column with different types across tables.
type ColumnTypeDiff struct {
	Name      string
	LeftType  string
	RightType string
}

// VolumeResult compares row counts.
type VolumeResult struct {
	Match      bool
	LeftCount  int64
	RightCount int64
}

func (v VolumeResult) String() string {
	if v.Match {
		return fmt.Sprintf("MATCH (%d rows)", v.LeftCount)
	}

	return fmt.Sprintf("MISMATCH (left=%d, right=%d, delta=%d)",
		v.LeftCount, v.RightCount, v.RightCount-v.LeftCount)
}

// ContentResult reports row-level differences.
type ContentResult struct {
	Match     bool
	LeftOnly  int64  // rows only in left.
	RightOnly int64  // rows only in right.
	Differed  int64  // rows present in both but with different values.
	Matched   int64  // rows identical in both.
	Diffs     []Diff // first N actual diffs for inspection.
}

// Diff is a single row-level difference.
type Diff struct {
	Key    string
	Side   string            // "left-only", "right-only", "both".
	Fields map[string][2]any // field → [left, right] values (only for "both").
}

// StatsResult compares column-level aggregates.
type StatsResult struct {
	Match bool
	Left  []ColumnStat
	Right []ColumnStat
	Diffs []StatDiff
}

// ColumnStat holds aggregates for a numeric column.
type ColumnStat struct {
	Column        string
	Sum           float64
	Avg           float64
	Min           float64
	Max           float64
	CountDistinct int64
	NullCount     int64
}

// StatDiff records a column where stats differ between left and right.
type StatDiff struct {
	Column string
	Field  string // "Sum", "Avg", etc.
	Left   float64
	Right  float64
	Delta  float64
}
