package sloggcp_test

import (
	"context"
	"io"
	"testing"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"

	"log/slog"
)

func BenchmarkHandler_Handle(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, func(_ context.Context) (string, string, bool) {
		return "abc123", "def456", true
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
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})
	h := sloggcp.NewHandler(inner, func(_ context.Context) (string, string, bool) {
		return "abc123def456", "00000000deadbeef", true
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
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, func(_ context.Context) (string, string, bool) {
		return "abc", "def", false
	}, "proj", sloggcp.WithEventID(false))

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "no event id") //nolint:sloglint // Benchmark format.
	}
}

func BenchmarkHandler_NoResolver(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, nil)
	h := sloggcp.NewHandler(inner, nil, "proj")

	ctx := context.Background()
	logger := slog.New(h)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "no resolver") //nolint:sloglint // Benchmark format.
	}
}
