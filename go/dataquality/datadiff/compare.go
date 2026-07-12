package datadiff

import (
	"context"
	"fmt"
	"math"
)

// Option configures a comparison run.
type Option func(*options)

type options struct {
	maxDiffs       int     // cap on row-level diffs returned (0 = unlimited).
	statsTolerance float64 // relative tolerance for stats comparison (0 = exact).
}

// WithMaxDiffs caps the number of row-level diffs returned.
func WithMaxDiffs(n int) Option {
	return func(o *options) { o.maxDiffs = n }
}

// WithStatsTolerance sets the relative tolerance for stats comparison.
// A tolerance of 0.01 means 1% relative difference is acceptable.
func WithStatsTolerance(t float64) Option {
	return func(o *options) { o.statsTolerance = t }
}

// Config specifies what to compare.
type Config struct {
	Key    string // primary key column for joining rows.
	Filter string // optional WHERE clause (e.g. partition filter).
}

// Reconciler orchestrates comparison layers through a Comparator port.
type Reconciler struct {
	cmp Comparator
}

// NewReconciler creates a reconciler with the given comparator.
func NewReconciler(cmp Comparator) *Reconciler {
	return &Reconciler{cmp: cmp}
}

// Compare runs all four comparison layers.
// Each layer is independent — a failure in one does not stop the others.
func (r *Reconciler) Compare(ctx context.Context, cfg Config, opts ...Option) (Result, error) {
	const (
		defaultMaxDiffs       = 100
		defaultStatsTolerance = 1e-9
	)

	o := options{maxDiffs: defaultMaxDiffs, statsTolerance: defaultStatsTolerance} // tiny default tolerance for FP noise.
	for _, opt := range opts {
		opt(&o)
	}

	result := Result{
		Spec: ComparisonSpec{
			LeftName:  r.cmp.Left(),
			RightName: r.cmp.Right(),
			Key:       cfg.Key,
			Filter:    cfg.Filter,
		},
	}
	var firstErr error

	// Layer 1: Schema.
	schema, err := r.cmp.CompareSchema(ctx)
	if err != nil {
		firstErr = fmt.Errorf("schema: %w", err)
	}
	result.Schema = schema

	// Layer 2: Volume.
	volume, err := r.cmp.CompareVolume(ctx, cfg.Filter)
	if err != nil && firstErr == nil {
		firstErr = fmt.Errorf("volume: %w", err)
	}
	result.Volume = volume

	// Layer 3: Content (only if schema matches — mismatched columns break diffs).
	if result.Schema.Match {
		content, err := r.cmp.CompareContent(ctx, cfg.Key, cfg.Filter, o.maxDiffs)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("content: %w", err)
		}
		result.Content = content
	}

	// Layer 4: Stats (apply tolerance in domain).
	stats, err := r.cmp.CompareStats(ctx, cfg.Filter)
	if err != nil && firstErr == nil {
		firstErr = fmt.Errorf("stats: %w", err)
	}
	applyStatsTolerance(&stats, o.statsTolerance)
	result.Stats = stats

	return result, firstErr
}

func applyStatsTolerance(s *StatsResult, tolerance float64) {
	if tolerance == 0 || len(s.Diffs) == 0 {
		return
	}

	n := 0
	for _, d := range s.Diffs {
		base := math.Max(math.Abs(d.Left), math.Abs(d.Right))
		if base == 0 {
			s.Diffs[n] = d
			n++

			continue
		}
		if math.Abs(d.Delta)/base > tolerance {
			s.Diffs[n] = d
			n++
		}
	}
	s.Diffs = s.Diffs[:n]
	s.Match = n == 0
}
