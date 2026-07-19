// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package otelgcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
	"github.com/duizendstra/alexandria/go/slog-gcp/otelgcp"
)

// spanContext builds a deterministic OpenTelemetry span context. In
// production the OpenTelemetry SDK places the active span in the context.
func spanContext() context.Context {
	traceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	spanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(context.Background(), sc)
}

// ExampleNewResolver wires the OpenTelemetry bridge into a slog-gcp
// handler so every log line carries the Cloud Logging trace fields.
func ExampleNewResolver() {
	var buf bytes.Buffer

	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})

	// The resolver extracts the active OpenTelemetry span from the context
	// of each log call. In production: slog.SetDefault(slog.New(handler)).
	handler := sloggcp.NewHandler(inner, otelgcp.NewResolver(), "my-project")
	logger := slog.New(handler)

	logger.InfoContext(spanContext(), "processing request")

	var entry map[string]any
	_ = json.Unmarshal(buf.Bytes(), &entry)

	fmt.Println(entry["message"])
	fmt.Println(entry[sloggcp.FieldTrace])
	fmt.Println(entry[sloggcp.FieldSpanID])
	fmt.Println(entry[sloggcp.FieldTraceSampled])
	// Output:
	// processing request
	// projects/my-project/traces/0af7651916cd43dd8448eb211c80319c
	// b7ad6b7169203331
	// true
}

// ExampleNewResolver_direct calls the resolver directly to show the
// TraceContext it derives from an OpenTelemetry span context.
func ExampleNewResolver_direct() {
	resolver := otelgcp.NewResolver()

	tc := resolver(spanContext())

	fmt.Println(tc.TraceID)
	fmt.Println(tc.SpanID)
	fmt.Println(tc.Sampled)
	// Output:
	// 0af7651916cd43dd8448eb211c80319c
	// b7ad6b7169203331
	// true
}
