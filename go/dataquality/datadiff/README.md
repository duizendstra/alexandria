# go/dataquality/datadiff

`go/dataquality/datadiff` provides a schema-based data verification and statistical difference engine for comparing structured tables across different systems or environments.

## Features

- **Multi-Layered Reconciliation**: Four independent comparison layers: Schema validation, Volume (row counts), Content hash comparison, and Column-level statistical aggregates.
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
