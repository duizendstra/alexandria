package datadiff

import "context"

// Comparator is the port for comparing two datasets.
//
// For same-platform (BQ-to-BQ): one adapter does server-side cross-project SQL.
// For cross-platform: adapter wraps two DataSources and compares client-side.
//
// The domain orchestrates layers and decides pass/fail.
// The adapter decides HOW to compare efficiently.
type Comparator interface {
	// Left returns a human-readable name for the left side.
	Left() string

	// Right returns a human-readable name for the right side.
	Right() string

	// CompareSchema compares column structure between the two tables.
	CompareSchema(ctx context.Context) (SchemaResult, error)

	// CompareVolume compares row counts.
	CompareVolume(ctx context.Context, filter string) (VolumeResult, error)

	// CompareContent compares row-level content using hashes.
	// Returns up to maxDiffs detailed differences.
	CompareContent(ctx context.Context, key string, filter string, maxDiffs int) (ContentResult, error)

	// CompareStats compares column-level aggregates for numeric columns.
	CompareStats(ctx context.Context, filter string) (StatsResult, error)
}
