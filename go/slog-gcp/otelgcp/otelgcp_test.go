// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package otelgcp_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/duizendstra/alexandria/go/slog-gcp/otelgcp"
)

func TestNewResolver(t *testing.T) {
	t.Parallel()

	const (
		wantTraceID = "0af7651916cd43dd8448eb211c80319c"
		wantSpanID  = "b7ad6b7169203331"
	)

	resolver := otelgcp.NewResolver()

	t.Run("empty context", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tc := resolver(ctx)
		if tc.TraceID != "" || tc.SpanID != "" || tc.Sampled {
			t.Errorf("expected empty TraceContext, got %+v", tc)
		}
	})

	t.Run("valid span context", func(t *testing.T) {
		t.Parallel()
		traceID, err := trace.TraceIDFromHex(wantTraceID)
		if err != nil {
			t.Fatalf("failed to parse trace ID: %v", err)
		}
		spanID, err := trace.SpanIDFromHex(wantSpanID)
		if err != nil {
			t.Fatalf("failed to parse span ID: %v", err)
		}

		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})

		ctx := trace.ContextWithSpanContext(context.Background(), sc)
		tc := resolver(ctx)

		if tc.TraceID != wantTraceID {
			t.Errorf("expected trace ID %q, got %q", wantTraceID, tc.TraceID)
		}
		if tc.SpanID != wantSpanID {
			t.Errorf("expected span ID %q, got %q", wantSpanID, tc.SpanID)
		}
		if !tc.Sampled {
			t.Error("expected sampled to be true")
		}
	})

	t.Run("invalid span context", func(t *testing.T) {
		t.Parallel()
		// Invalid span context has zero values for TraceID and SpanID.
		sc := trace.NewSpanContext(trace.SpanContextConfig{})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)
		tc := resolver(ctx)

		if tc.TraceID != "" || tc.SpanID != "" || tc.Sampled {
			t.Errorf("expected empty TraceContext, got %+v", tc)
		}
	})
}
