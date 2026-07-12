// Package datadiff proves two datasets are equivalent through layered comparison.
//
// BC:      DataQuality
// Concern: How do we prove two datasets are equivalent?
//
// A 4-layer comparison framework that checks schema, volume, content, and
// statistical aggregates between two tables. The [Reconciler] orchestrates
// these layers through the [Comparator] port, returning a structured [Result]
// with per-layer pass/fail and detailed diffs for inspection.
//
// Use this package during data migrations to gate environment promotions
// (dev → acc → prod). Manual spot checks miss edge cases — automated,
// repeatable, multi-layer comparison ensures cutover decisions are based
// on evidence, not hope.
//
// The four layers:
//
//  1. Schema  — column names, types, and nesting
//  2. Volume  — row counts
//  3. Content — row-level hash comparison (skipped if schema mismatches)
//  4. Stats   — column-level aggregates (sum, avg, min, max, distinct, nulls)
//
// Each layer is independent — a failure in one does not stop the others.
// Tolerance logic (e.g. floating-point noise) is applied in the domain
// without any platform knowledge.
//
// Pure domain — zero external dependencies. Adapters (e.g. bqcompare for
// BigQuery) implement [Comparator] and live in platform/gcp/.
package datadiff
