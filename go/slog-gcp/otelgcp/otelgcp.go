// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

// Package otelgcp provides OpenTelemetry integration for slog-gcp.
// It bridges OpenTelemetry trace contexts into the slog-gcp TraceContext
// format required for Cloud Logging correlation.
package otelgcp

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

// NewResolver returns an sloggcp.IDResolver that extracts W3C traceparent
// contexts from the OpenTelemetry span in the current context.
//
// This allows slog-gcp to correlate logs with distributed traces managed
// by OpenTelemetry (e.g., on GKE or generic GCP environments).
//
//	handler := sloggcp.NewHandler(inner, otelgcp.NewResolver(), "my-gcp-project")
//	slog.SetDefault(slog.New(handler))
func NewResolver() sloggcp.IDResolver {
	type cache struct {
		traceID      trace.TraceID
		spanID       trace.SpanID
		sampled      bool
		traceContext sloggcp.TraceContext
	}
	var lastCache atomic.Pointer[cache]

	return func(ctx context.Context) sloggcp.TraceContext {
		if ctx == nil {
			return sloggcp.TraceContext{}
		}
		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()
		if !sc.IsValid() {
			return sloggcp.TraceContext{}
		}

		if c := lastCache.Load(); c != nil && c.traceID == sc.TraceID() && c.spanID == sc.SpanID() && c.sampled == sc.IsSampled() {
			return c.traceContext
		}

		tc := sloggcp.TraceContext{
			TraceID: sc.TraceID().String(),
			SpanID:  sc.SpanID().String(),
			Sampled: sc.IsSampled(),
		}
		lastCache.Store(&cache{
			traceID:      sc.TraceID(),
			spanID:       sc.SpanID(),
			sampled:      sc.IsSampled(),
			traceContext: tc,
		})

		return tc
	}
}
