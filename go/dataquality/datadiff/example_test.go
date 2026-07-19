// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

//nolint:goconst // example schemas contain repetitive standard database field types
package datadiff_test

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

// memComparator is a minimal in-memory Comparator used to demonstrate the
// Reconciler offline. Real adapters (e.g. bqcompare for BigQuery) run the
// comparison server-side.
type memComparator struct {
	leftCols, rightCols   []datadiff.Column
	leftRows, rightRows   int64
	leftStats, rightStats []datadiff.ColumnStat
}

func (c *memComparator) Left() string  { return "prod-project.sales.orders" }
func (c *memComparator) Right() string { return "dev-project.sales.orders" }

func (c *memComparator) CompareSchema(_ context.Context) (datadiff.SchemaResult, error) {
	return datadiff.DiffColumns(c.leftCols, c.rightCols), nil
}

func (c *memComparator) CompareVolume(_ context.Context, _ string) (datadiff.VolumeResult, error) {
	return datadiff.VolumeResult{
		Match:      c.leftRows == c.rightRows,
		LeftCount:  c.leftRows,
		RightCount: c.rightRows,
	}, nil
}

func (c *memComparator) CompareContent(_ context.Context, _, _ string, _ int) (datadiff.ContentResult, error) {
	return datadiff.ContentResult{Match: true, Matched: c.leftRows}, nil
}

func (c *memComparator) CompareStats(_ context.Context, _ string) (datadiff.StatsResult, error) {
	return datadiff.DiffStats(c.leftStats, c.rightStats, 0), nil
}

func ExampleParseTarget() {
	target, err := datadiff.ParseTarget("my-project.sales.orders")
	if err != nil {
		fmt.Println("parse:", err)

		return
	}

	fmt.Println(target.Project, target.Dataset, target.Table)
	fmt.Println(target.FullyQualified())
	// Output:
	// my-project sales orders
	// `my-project.sales.orders`
}

func ExampleParseTargetPair() {
	left, right, err := datadiff.ParseTargetPair(
		"prod-project.sales.orders",
		"dev-project.sales.orders",
	)
	if err != nil {
		fmt.Println("parse:", err)

		return
	}

	fmt.Println("left: ", left)
	fmt.Println("right:", right)
	// Output:
	// left:  prod-project.sales.orders
	// right: dev-project.sales.orders
}

func ExampleReconciler_Compare() {
	// Both sides share the same schema, row count, and aggregates,
	// so every layer matches.
	cols := []datadiff.Column{
		{Name: "id", DataType: "INT64", Mode: datadiff.ModeRequired, Position: 1},
		{Name: "amount", DataType: "FLOAT64", Mode: datadiff.ModeNullable, Position: 2},
	}
	stats := []datadiff.ColumnStat{{Column: "amount", Sum: 60.75, Avg: 20.25, Min: 9.5, Max: 31.0}}

	cmp := &memComparator{
		leftCols: cols, rightCols: cols,
		leftRows: 3, rightRows: 3,
		leftStats: stats, rightStats: stats,
	}

	r := datadiff.NewReconciler(cmp)
	result, err := r.Compare(context.Background(),
		datadiff.Config{Key: "id"},
		datadiff.WithMaxDiffs(10),
		datadiff.WithStatsTolerance(0.001),
	)
	if err != nil {
		fmt.Println("compare:", err)

		return
	}

	fmt.Println("schema: ", result.Schema.Match)
	fmt.Println("volume: ", result.Volume)
	fmt.Println("content:", result.Content.Match)
	fmt.Println("stats:  ", result.Stats.Match)
	fmt.Println("pass:   ", result.Pass())
	// Output:
	// schema:  true
	// volume:  MATCH (3 rows)
	// content: true
	// stats:   true
	// pass:    true
}

func ExampleDiffColumns() {
	left := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "email", DataType: "STRING"},
	}
	right := []datadiff.Column{
		{Name: "id", DataType: "STRING"},
		{Name: "phone", DataType: "STRING"},
	}

	res := datadiff.DiffColumns(left, right)

	fmt.Println("match:", res.Match)
	for _, c := range res.LeftOnly {
		fmt.Println("left-only: ", c.Name)
	}
	for _, c := range res.RightOnly {
		fmt.Println("right-only:", c.Name)
	}
	for _, d := range res.TypeDiffs {
		fmt.Printf("type-diff:  %s (%s -> %s)\n", d.Name, d.LeftType, d.RightType)
	}
	// Output:
	// match: false
	// left-only:  email
	// right-only: phone
	// type-diff:  id (INT64 -> STRING)
}

func ExampleDiffStats() {
	left := []datadiff.ColumnStat{{Column: "amount", Sum: 1000, Avg: 10}}
	right := []datadiff.ColumnStat{{Column: "amount", Sum: 1005, Avg: 10.05}}

	// Exact comparison flags every difference.
	exact := datadiff.DiffStats(left, right, 0)
	fmt.Println("exact match:", exact.Match)
	for _, d := range exact.Diffs {
		fmt.Printf("%s.%s: left=%.2f right=%.2f delta=%.2f\n",
			d.Column, d.Field, d.Left, d.Right, d.Delta)
	}

	// A 1% relative tolerance absorbs the 0.5% drift.
	tolerant := datadiff.DiffStats(left, right, 0.01)
	fmt.Println("tolerant match:", tolerant.Match)
	// Output:
	// exact match: false
	// amount.Sum: left=1000.00 right=1005.00 delta=5.00
	// amount.Avg: left=10.00 right=10.05 delta=0.05
	// tolerant match: true
}

func ExampleFlattenAll() {
	cols := []datadiff.Column{
		{Name: "id", DataType: "INT64", Mode: datadiff.ModeRequired},
		{Name: "address", DataType: "RECORD", Mode: datadiff.ModeNullable, Children: []datadiff.Column{
			{Name: "city", DataType: "STRING", Mode: datadiff.ModeNullable},
			{Name: "zip", DataType: "STRING", Mode: datadiff.ModeNullable},
		}},
	}

	for _, c := range datadiff.FlattenAll(cols) {
		fmt.Printf("%s (%s)\n", c.Name, c.DataType)
	}
	// Output:
	// id (INT64)
	// address.city (STRING)
	// address.zip (STRING)
}

func ExamplePrintResult() {
	result := datadiff.Result{
		Schema:  datadiff.SchemaResult{Match: true},
		Volume:  datadiff.VolumeResult{Match: false, LeftCount: 1000, RightCount: 998},
		Content: datadiff.ContentResult{Match: true, Matched: 998},
		Stats:   datadiff.StatsResult{Match: true},
	}

	datadiff.PrintResult(&result)
	// Output:
	// ✅ Schema: MATCH
	// ❌ Volume: MISMATCH (left=1000, right=998, delta=-2)
	// ✅ Content: MATCH (998 rows identical)
	// ✅ Stats: MATCH
	//
	// ⚠️  COMPARISON FAILED
}

func ExampleVolumeResult_String() {
	v := datadiff.VolumeResult{Match: false, LeftCount: 1000, RightCount: 998}
	fmt.Println(v)
	// Output:
	// MISMATCH (left=1000, right=998, delta=-2)
}

func ExampleTarget_FullyQualified() {
	target := datadiff.Target{Project: "my-project", Dataset: "sales", Table: "orders"}
	fmt.Println(target.FullyQualified())
	// Output:
	// `my-project.sales.orders`
}
