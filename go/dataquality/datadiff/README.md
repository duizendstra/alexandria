# go/dataquality/datadiff

`go/dataquality/datadiff` provides a schema-based data verification and statistical difference engine for comparing structured tables across different systems or environments.

## Features

- **Multi-Layered Table Reconciliation**: Four independent comparison layers: Schema validation, Volume (row counts), Content hash comparison, and Column-level statistical aggregates.
- **In-Memory Set & Map Parity Differ**: Generic `SetDiff[T]`, `MapDiff[K, V]`, and 3-way migration reconciler `ThreeWayDiff[T]` (`baseline ∖ leftovers == target`) for fast memory-level manifest diffing.
- **Port/Adapter Architecture**: Zero external dependencies in the core domain; platform-specific comparators (e.g., BigQuery comparison) are decoupled.
- **Nested Column Flattening**: Recursive dot-notation flattener for record/struct data structures.
- **Configurable Diffs Cap**: Prevents memory bloat by capping the maximum returned row-level differences.
- **Stats Relative Tolerance**: Allows float comparison with customizable relative error margins (e.g., within 1%).

## Installation

```bash
go get github.com/duizendstra/alexandria/go/dataquality/datadiff
```

## Quick Start

### Implementing and Running a Comparison Reconciler

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

// mockComparator implements datadiff.Comparator for demonstration purposes
type mockComparator struct{}

func (m mockComparator) Left() string  { return "production_users" }
func (m mockComparator) Right() string { return "replica_users" }

func (m mockComparator) CompareSchema(ctx context.Context) (datadiff.SchemaResult, error) {
	return datadiff.SchemaResult{Match: true}, nil
}

func (m mockComparator) CompareVolume(ctx context.Context, filter string) (datadiff.VolumeResult, error) {
	return datadiff.VolumeResult{Match: true, LeftCount: 1500, RightCount: 1500}, nil
}

func (m mockComparator) CompareContent(ctx context.Context, key, filter string, maxDiffs int) (datadiff.ContentResult, error) {
	return datadiff.ContentResult{Match: true}, nil
}

func (m mockComparator) CompareStats(ctx context.Context, filter string) (datadiff.StatsResult, error) {
	return datadiff.StatsResult{Match: true}, nil
}

func main() {
	ctx := context.Background()
	cmp := mockComparator{}
	reconciler := datadiff.NewReconciler(cmp)

	// Run comparison across all 4 verification layers
	result, err := reconciler.Compare(ctx, datadiff.Config{
		Key:    "user_id",
		Filter: "status = 'active'",
	}, datadiff.WithMaxDiffs(10), datadiff.WithStatsTolerance(0.01))

	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	fmt.Printf("Reconciliation match status: %t (Schema Match: %t, Volume Match: %t)\n",
		result.Pass(), result.Schema.Match, result.Volume.Match)
}
```

## SRE & Performance Hardening details

1. **Zero-Allocation Struct Flattening**: The nested column flattening routine (`FlattenTo`) appends leaf columns directly into a pre-allocated accumulator slice rather than allocating new arrays during recursion, minimizing heap allocations.
2. **Memory Exhaustion Safeguard**: High-volume content discrepancies can consume significant memory. The `WithMaxDiffs` option enforces a hard cap on retrieved differences, safeguarding service memory from out-of-memory (OOM) failures.
3. **Execution Isolation**: Individual comparison layers execute independently. If content comparison errors out (e.g., due to temporary network timeouts), volume and schema analysis still complete, preserving observability of the data quality pipeline.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Migration verifiers**: work proving a rebuilt table matches the one it
  replaces before the old one is dropped.
- **Scheduled reconciliation**: recurring checks that two pipelines producing
  the same dataset have not drifted.

### Load-Bearing Promises
1. **Layers Short-Circuit In Order**: a schema mismatch skips the content
   layer rather than comparing rows that cannot correspond. A cheap disproof
   is never paid for with an expensive one.
2. **Every Difference Is Reported, Not The First**: left-only columns,
   right-only columns, type mismatches and nested record differences are all
   returned together, so one run gives the whole picture.
3. **Tolerance Is Honoured, Including Non-Finite**: statistical comparison
   respects the configured tolerance, and non-finite values are handled
   explicitly rather than propagating a silent `NaN` verdict.
4. **Failure Is Structured, Not An Abort**: when every layer fails the result
   still describes what failed at each layer.
5. **The Spec Is Captured In The Result**: a result records the comparison it
   came from, so a stored verdict remains interpretable later.
