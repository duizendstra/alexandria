// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp_test

import (
	"context"
	"io"
	"testing"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"

	"log/slog"
)

func BenchmarkHandler_Handle(b *testing.B) {
	//nolint:sloglint // Benchmark requires JSON formatting to io.Discard.
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, func(_ context.Context) sloggcp.TraceContext {
		return sloggcp.TraceContext{TraceID: "abc123", SpanID: "def456", Sampled: true}
	}, "bench-project")

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "benchmark message", "key", "value") //nolint:sloglint // Benchmark format.
	}
}

func BenchmarkFullChain_Handle(b *testing.B) {
	//nolint:sloglint // Benchmark requires JSON formatting to io.Discard.
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})
	h := sloggcp.NewHandler(inner, func(_ context.Context) sloggcp.TraceContext {
		return sloggcp.TraceContext{TraceID: "abc123def456", SpanID: "00000000deadbeef", Sampled: true}
	}, "bench-project")

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "benchmark message", "key", "value") //nolint:sloglint // Benchmark format.
	}
}

func BenchmarkHandler_WithEventID_Disabled(b *testing.B) {
	//nolint:sloglint // Benchmark requires JSON formatting to io.Discard.
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, func(_ context.Context) sloggcp.TraceContext {
		return sloggcp.TraceContext{TraceID: "abc", SpanID: "def", Sampled: false}
	}, "proj", sloggcp.WithEventID(false))

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "no event id")
	}
}

func BenchmarkHandler_NoResolver(b *testing.B) {
	//nolint:sloglint // Benchmark requires JSON formatting to io.Discard.
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, nil, "proj")

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "no resolver")
	}
}

