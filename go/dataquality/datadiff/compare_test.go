//nolint:goconst // test schemas contain repetitive standard database field types
package datadiff_test

import (
	"context"
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

// stubComparator implements Comparator for testing domain logic.
type stubComparator struct {
	schema  datadiff.SchemaResult
	volume  datadiff.VolumeResult
	content datadiff.ContentResult
	stats   datadiff.StatsResult
}

func (s *stubComparator) Left() string  { return "left-project.ds.t" }
func (s *stubComparator) Right() string { return "right-project.ds.t" }

func (s *stubComparator) CompareSchema(_ context.Context) (datadiff.SchemaResult, error) {
	return s.schema, nil
}

func (s *stubComparator) CompareVolume(_ context.Context, _ string) (datadiff.VolumeResult, error) {
	return s.volume, nil
}

func (s *stubComparator) CompareContent(_ context.Context, _, _ string, _ int) (datadiff.ContentResult, error) {
	return s.content, nil
}

func (s *stubComparator) CompareStats(_ context.Context, _ string) (datadiff.StatsResult, error) {
	return s.stats, nil
}

func TestCompare_IdenticalTables(t *testing.T) {
	cmp := &stubComparator{
		schema:  datadiff.SchemaResult{Match: true},
		volume:  datadiff.VolumeResult{Match: true, LeftCount: 1000, RightCount: 1000},
		content: datadiff.ContentResult{Match: true, Matched: 1000},
		stats:   datadiff.StatsResult{Match: true},
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass() {
		t.Error("identical tables should pass")
	}
}

func TestCompare_SchemaMismatch_SkipsContent(t *testing.T) {
	cmp := &stubComparator{
		schema: datadiff.SchemaResult{
			Match:    false,
			LeftOnly: []datadiff.Column{{Name: "legacy_field", DataType: "STRING"}},
		},
		volume: datadiff.VolumeResult{Match: true, LeftCount: 100, RightCount: 100},
		// Content should NOT be called when schema mismatches.
		content: datadiff.ContentResult{Match: true, Matched: 9999},
		stats:   datadiff.StatsResult{Match: true},
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass() {
		t.Error("schema mismatch should not pass")
	}
	// Content should be zero-value (skipped), not the stub's value.
	if result.Content.Matched == 9999 {
		t.Error("content comparison should be skipped when schema mismatches")
	}
}

func TestCompare_VolumeMismatch(t *testing.T) {
	cmp := &stubComparator{
		schema: datadiff.SchemaResult{Match: true},
		volume: datadiff.VolumeResult{Match: false, LeftCount: 1000, RightCount: 999},
		content: datadiff.ContentResult{
			Match:    false,
			LeftOnly: 1,
		},
		stats: datadiff.StatsResult{Match: true},
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass() {
		t.Error("volume mismatch should not pass")
	}
	if result.Volume.LeftCount != 1000 || result.Volume.RightCount != 999 {
		t.Errorf("counts wrong: %d vs %d", result.Volume.LeftCount, result.Volume.RightCount)
	}
}

func TestCompare_ContentDiffs(t *testing.T) {
	cmp := &stubComparator{
		schema: datadiff.SchemaResult{Match: true},
		volume: datadiff.VolumeResult{Match: true, LeftCount: 100, RightCount: 100},
		content: datadiff.ContentResult{
			Match:    false,
			LeftOnly: 1, RightOnly: 1, Differed: 1, Matched: 97,
			Diffs: []datadiff.Diff{
				{Key: "42", Side: "both", Fields: map[string][2]any{"price": {9.99, 10.99}}},
				{Key: "99", Side: "left-only"},
				{Key: "100", Side: "right-only"},
			},
		},
		stats: datadiff.StatsResult{Match: true},
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content.Match {
		t.Error("content should not match")
	}
	if result.Content.Differed != 1 {
		t.Errorf("differed = %d, want 1", result.Content.Differed)
	}
}

func TestCompare_StatsTolerance(t *testing.T) {
	cmp := &stubComparator{
		schema:  datadiff.SchemaResult{Match: true},
		volume:  datadiff.VolumeResult{Match: true, LeftCount: 100, RightCount: 100},
		content: datadiff.ContentResult{Match: true, Matched: 100},
		stats: datadiff.StatsResult{
			Match: false,
			Diffs: []datadiff.StatDiff{
				{Column: "amount", Field: "Sum", Left: 1000.0, Right: 1005.0, Delta: 5.0}, // 0.5%
			},
		},
	}

	r := datadiff.NewReconciler(cmp)

	// Without tolerance — should flag the diff.
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Stats.Match {
		t.Error("stats should not match without tolerance")
	}

	// With 1% tolerance — 0.5% diff should pass.
	result, err = r.Compare(context.Background(), datadiff.Config{Key: "id"},
		datadiff.WithStatsTolerance(0.01))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Stats.Match {
		t.Error("stats should match with 1%% tolerance")
	}
}

func TestCompare_AllLayersFail(t *testing.T) {
	cmp := &stubComparator{
		schema: datadiff.SchemaResult{
			Match:    false,
			LeftOnly: []datadiff.Column{{Name: "old", DataType: "STRING"}},
		},
		volume:  datadiff.VolumeResult{Match: false, LeftCount: 100, RightCount: 90},
		content: datadiff.ContentResult{Match: false, Differed: 10},
		stats: datadiff.StatsResult{
			Match: false,
			Diffs: []datadiff.StatDiff{{Column: "x", Field: "Sum", Left: 1, Right: 2, Delta: 1}},
		},
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(), datadiff.Config{Key: "id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass() {
		t.Error("should not pass when all layers fail")
	}
	if result.Schema.Match {
		t.Error("schema should not match")
	}
	if result.Volume.Match {
		t.Error("volume should not match")
	}
	// Content is skipped due to schema mismatch — should be zero-value.
	if result.Content.Differed != 0 {
		t.Error("content should be zero-value when schema mismatches")
	}
}

func TestCompare_SpecCapture(t *testing.T) {
	cmp := &stubComparator{
		schema:  datadiff.SchemaResult{Match: true},
		volume:  datadiff.VolumeResult{Match: true, LeftCount: 50, RightCount: 50},
		content: datadiff.ContentResult{Match: true, Matched: 50},
		stats:   datadiff.StatsResult{Match: true},
	}

	cfg := datadiff.Config{Key: "tx_key", Filter: "transaction_date < CURRENT_DATE()"}
	r := datadiff.NewReconciler(cmp)
	result, _ := r.Compare(context.Background(), cfg)

	if result.Spec.Key != "tx_key" {
		t.Errorf("spec.Key = %q, want %q", result.Spec.Key, "tx_key")
	}
	if result.Spec.Filter != "transaction_date < CURRENT_DATE()" {
		t.Errorf("spec.Filter = %q", result.Spec.Filter)
	}
	if result.Spec.LeftName != "left-project.ds.t" {
		t.Errorf("spec.LeftName = %q", result.Spec.LeftName)
	}
}
