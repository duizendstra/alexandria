// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
	sloggcptest "github.com/duizendstra/alexandria/go/slog-gcp/sloggcptest"

	"log/slog"
)

const (
	testSpanID      = "00000000deadbeef"
	levelDebugStr   = "DEBUG"
	levelWarningStr = "WARNING"
	testHealthURL   = "/api/health"
)

// testResolver returns a fixed IDResolver for testing.
func testResolver(traceID, spanID string, sampled bool) sloggcp.IDResolver {
	return func(_ context.Context) sloggcp.TraceContext {
		return sloggcp.TraceContext{
			TraceID: traceID,
			SpanID:  spanID,
			Sampled: sampled,
		}
	}
}

// --- Handler core tests ---.

func TestHandler_InjectsAllFields(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-abc", testSpanID, true), "my-project"))

	logger.InfoContext(context.Background(), "test message")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	entry := entries[0]

	if _, ok := entry["event_id"].(string); !ok {
		t.Error("event_id missing")
	}

	wantTrace := "projects/my-project/traces/trace-abc"
	if got := entry["logging.googleapis.com/trace"]; got != wantTrace {
		t.Errorf("trace = %v, want %v", got, wantTrace)
	}

	if got := entry["logging.googleapis.com/spanId"]; got != testSpanID {
		t.Errorf("spanId = %v, want %s", got, testSpanID)
	}

	if got := entry["logging.googleapis.com/trace_sampled"]; got != true {
		t.Errorf("trace_sampled = %v, want true", got)
	}
}

func TestHandler_EventIDUnique(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("t1", "", false), "proj"))

	logger.InfoContext(context.Background(), "first")
	logger.InfoContext(context.Background(), "second")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 2)

	if entries[0]["event_id"] == entries[1]["event_id"] {
		t.Error("event_id not unique across lines")
	}
}

func TestHandler_NoResolver(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, nil, "proj"))

	logger.InfoContext(context.Background(), "no resolver")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	if _, ok := entries[0]["event_id"]; !ok {
		t.Error("event_id should always be present")
	}

	if _, ok := entries[0]["logging.googleapis.com/trace"]; ok {
		t.Error("trace should not appear with nil resolver")
	}
}

func TestHandler_WithEventID_Disabled(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, nil, "", sloggcp.WithEventID(false)))

	logger.InfoContext(context.Background(), "no event id")

	entries := sloggcptest.LogEntries(buf)

	if _, ok := entries[0]["event_id"]; ok {
		t.Error("event_id should not be present when disabled")
	}
}

func TestHandler_EmptyIDs_NotInjected(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("", "", false), "proj"))

	logger.InfoContext(context.Background(), "empty")

	entries := sloggcptest.LogEntries(buf)

	if _, ok := entries[0]["logging.googleapis.com/trace"]; ok {
		t.Error("empty traceID should not produce trace field")
	}

	if _, ok := entries[0]["logging.googleapis.com/spanId"]; ok {
		t.Error("empty spanID should not produce spanId field")
	}

	if _, ok := entries[0]["logging.googleapis.com/trace_sampled"]; ok {
		t.Error("trace_sampled should not appear when traceID is empty")
	}
}

func TestHandler_TraceSampled_False(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), "proj"))

	logger.InfoContext(context.Background(), "sampled false")

	entries := sloggcptest.LogEntries(buf)

	if got := entries[0]["logging.googleapis.com/trace_sampled"]; got != false {
		t.Errorf("trace_sampled = %v, want false", got)
	}
}

func TestHandler_Enabled_Delegates(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	h := sloggcp.NewHandler(inner, nil, "proj")

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("handler should not be enabled at DEBUG when inner is WARN")
	}

	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("handler should be enabled at WARN")
	}

	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("handler should be enabled at ERROR")
	}
}

func TestHandler_Handle_ErrorPropagation(t *testing.T) {
	t.Parallel()

	errInner := errors.New("inner handler failed")
	h := sloggcp.NewHandler(&failHandler{err: errInner}, nil, "proj")

	err := h.Handle(context.Background(), slog.Record{})
	if !errors.Is(err, errInner) {
		t.Errorf("got err=%v, want %v", err, errInner)
	}
}

// failHandler always returns an error from Handle.
type failHandler struct {
	slog.Handler
	err error
}

func (f *failHandler) Enabled(context.Context, slog.Level) bool { return true }

func (f *failHandler) Handle(context.Context, slog.Record) error { return f.err }

func (f *failHandler) WithAttrs([]slog.Attr) slog.Handler { return f }

func (f *failHandler) WithGroup(string) slog.Handler { return f }

// --- WithAttrs / WithGroup ---.

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	h := sloggcp.NewHandler(inner, testResolver("t1", "s1", false), "proj")
	wrapped := h.WithAttrs([]slog.Attr{
		slog.String("service", "test"),
	})
	logger := slog.New(wrapped)

	logger.InfoContext(context.Background(), "with attrs")

	entries := sloggcptest.LogEntries(buf)

	if entries[0]["service"] != "test" {
		t.Error("pre-set attr not preserved")
	}

	if _, ok := entries[0]["event_id"]; !ok {
		t.Error("event_id missing after WithAttrs")
	}
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	h := sloggcp.NewHandler(inner, nil, "proj")
	grouped := h.WithGroup("req")
	logger := slog.New(grouped)

	logger.InfoContext(context.Background(), "grouped", "method", "GET") //nolint:sloglint // Test format.

	entries := sloggcptest.LogEntries(buf)

	if _, ok := entries[0]["req"]; !ok {
		t.Error("group not applied")
	}
}

// --- GCPReplaceAttr ---.

func TestGCPReplaceAttr_MessageKey(t *testing.T) {
	t.Parallel()

	msg := slog.Attr{Key: slog.MessageKey, Value: slog.StringValue("hello")}
	if got := sloggcp.GCPReplaceAttr(nil, msg); got.Key != "message" {
		t.Errorf("msg key = %q, want message", got.Key)
	}
}

func TestGCPReplaceAttr_SeverityMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug - 4, levelDebugStr},
		{slog.LevelDebug, levelDebugStr},
		{slog.LevelDebug + 2, levelDebugStr},
		{slog.LevelInfo, "INFO"},
		{slog.LevelInfo + 1, "INFO"},
		{slog.LevelInfo + 2, "NOTICE"},
		{slog.LevelInfo + 3, "NOTICE"},
		{slog.LevelWarn, levelWarningStr},
		{slog.LevelWarn + 1, levelWarningStr},
		{slog.LevelError, "ERROR"},
		{slog.LevelError + 3, "ERROR"},
		{slog.LevelError + 4, "CRITICAL"},
		{slog.LevelError + 7, "CRITICAL"},
		{slog.LevelError + 8, "ALERT"},
		{slog.LevelError + 11, "ALERT"},
		{slog.LevelError + 12, "EMERGENCY"},
		{slog.LevelError + 20, "EMERGENCY"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			t.Parallel()

			attr := slog.Attr{Key: slog.LevelKey, Value: slog.AnyValue(tc.level)}
			got := sloggcp.GCPReplaceAttr(nil, attr)

			if got.Key != "severity" {
				t.Errorf("key = %q, want severity", got.Key)
			}

			if got.Value.String() != tc.want {
				t.Errorf("severity = %q, want %q", got.Value.String(), tc.want)
			}
		})
	}
}

func TestGCPReplaceAttr_NonLevel_Passthrough(t *testing.T) {
	t.Parallel()

	attr := slog.String("custom", "value")
	got := sloggcp.GCPReplaceAttr(nil, attr)

	if got.Key != "custom" || got.Value.String() != "value" {
		t.Error("non-level attribute should pass through unchanged")
	}
}

func TestGCPReplaceAttr_TimestampMapping(t *testing.T) {
	t.Parallel()

	attr := slog.Attr{Key: slog.TimeKey, Value: slog.StringValue("2026-06-28T12:00:00Z")}
	got := sloggcp.GCPReplaceAttr(nil, attr)

	if got.Key != "timestamp" {
		t.Errorf("key = %q, want timestamp", got.Key)
	}
}

func TestGCPReplaceAttr_SourceLocationMapping(t *testing.T) {
	t.Parallel()

	source := &slog.Source{File: "main.go", Line: 42, Function: "main.main"}
	attr := slog.Attr{Key: slog.SourceKey, Value: slog.AnyValue(source)}
	got := sloggcp.GCPReplaceAttr(nil, attr)

	if got.Key != "logging.googleapis.com/sourceLocation" {
		t.Errorf("key = %q, want logging.googleapis.com/sourceLocation", got.Key)
	}
}

// --- parseCloudTraceHeader (via integration) ---.

func TestParseCloudTraceHeader_FullHeader(t *testing.T) {
	t.Parallel()

	// Span 456 decimal = 0x1c8 → "00000000000001c8".
	testTraceViaMiddleware(t,
		"abc123/456;o=1",
		"abc123",
		"00000000000001c8",
		true,
	)
}

func TestParseCloudTraceHeader_NoSpan(t *testing.T) {
	t.Parallel()

	testTraceViaMiddleware(t,
		"abc123;o=1",
		"abc123",
		"",
		true,
	)
}

func TestParseCloudTraceHeader_EmptyHeader(t *testing.T) {
	t.Parallel()

	// Empty header: middleware doesn't set context, so no trace fields.
	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, func(ctx context.Context) sloggcp.TraceContext {
		info := sloggcp.ParseCloudTraceHeaderForTest("")

		return sloggcp.TraceContext(info)
	}, "test-project"))

	logger.InfoContext(context.Background(), "empty header")

	entries := sloggcptest.LogEntries(buf)
	if len(entries) == 0 {
		t.Fatal("no log entries")
	}

	entry := entries[0]

	if _, ok := entry["logging.googleapis.com/trace"]; ok {
		t.Error("trace should not appear with empty header")
	}

	if _, ok := entry["logging.googleapis.com/spanId"]; ok {
		t.Error("spanId should not appear with empty header")
	}

	if _, ok := entry["logging.googleapis.com/trace_sampled"]; ok {
		t.Error("trace_sampled should not appear with empty traceID")
	}
}

func TestParseCloudTraceHeader_NoSampled(t *testing.T) {
	t.Parallel()

	testTraceViaMiddleware(t,
		"abc123/456",
		"abc123",
		"00000000000001c8",
		false,
	)
}

func TestParseCloudTraceHeader_SampledZero(t *testing.T) {
	t.Parallel()

	testTraceViaMiddleware(t,
		"abc123/456;o=0",
		"abc123",
		"00000000000001c8",
		false,
	)
}

func TestParseCloudTraceHeader_MultiSemicolon(t *testing.T) {
	t.Parallel()

	testTraceViaMiddleware(t,
		"abc123/456;o=1;extra=data",
		"abc123",
		"00000000000001c8",
		true,
	)
}

func TestParseCloudTraceHeader_HexSpanPassthrough(t *testing.T) {
	t.Parallel()

	// When span is already hex (not decimal), it passes through.
	testTraceViaMiddleware(t,
		"abc123/deadbeefcafe;o=1",
		"abc123",
		"deadbeefcafe", // Not a valid decimal, so returned as-is.
		true,
	)
}

type captureHandler struct {
	ctx context.Context
}

func (c *captureHandler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	c.ctx = r.Context()
}

// testTraceViaMiddleware exercises the full middleware→resolver→handler chain
// and asserts on the resulting log entry.
//nolint:unparam // Parameter flexibility useful for future test cases.
func testTraceViaMiddleware(t *testing.T, header, wantTraceID, wantSpanID string, wantSampled bool) {
	t.Helper()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)

	var capture captureHandler

	traced := sloggcp.TraceMiddleware(&capture)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)

	if header != "" {
		req.Header.Set("X-Cloud-Trace-Context", header)
	}

	traced.ServeHTTP(httptest.NewRecorder(), req)

	captured := capture.ctx

	// Now create a resolver that reads from context (mimicking InitCloudRun's resolver).
	resolver := func(ctx context.Context) sloggcp.TraceContext {
		traceHeader, _ := ctx.Value(sloggcp.TraceHeaderKeyType{}).(string)
		if traceHeader == "" {
			return sloggcp.TraceContext{}
		}

		info := sloggcp.ParseCloudTraceHeaderForTest(traceHeader)

		return sloggcp.TraceContext(info)
	}

	// Use our captured context to log.
	if captured == nil {
		// Empty header → no context stored, nothing to assert on trace fields.
		_ = inner
		_ = resolver

		return
	}

	logger := slog.New(sloggcp.NewHandler(inner, resolver, "test-project"))
	logger.InfoContext(captured, "trace test")

	entries := sloggcptest.LogEntries(buf)
	if len(entries) == 0 {
		t.Fatal("no log entries")
	}

	entry := entries[0]

	if wantTraceID != "" {
		wantFull := "projects/test-project/traces/" + wantTraceID
		if got := entry["logging.googleapis.com/trace"]; got != wantFull {
			t.Errorf("trace = %v, want %v", got, wantFull)
		}
	}

	if wantSpanID != "" {
		if got := entry["logging.googleapis.com/spanId"]; got != wantSpanID {
			t.Errorf("spanId = %v, want %v", got, wantSpanID)
		}
	}

	if got := entry["logging.googleapis.com/trace_sampled"]; got != wantSampled {
		t.Errorf("trace_sampled = %v, want %v", got, wantSampled)
	}
}

// --- TraceMiddleware ---.

type captureHeaderHandler struct {
	gotHeader string
}

func (c *captureHeaderHandler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	c.gotHeader, _ = r.Context().Value(sloggcp.TraceHeaderKeyType{}).(string)
}

func TestTraceMiddleware_SetsContext(t *testing.T) {
	t.Parallel()

	var capture captureHeaderHandler

	traced := sloggcp.TraceMiddleware(&capture)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Cloud-Trace-Context", "abc/123;o=1")
	traced.ServeHTTP(httptest.NewRecorder(), req)

	if capture.gotHeader != "abc/123;o=1" {
		t.Errorf("context header = %q, want abc/123;o=1", capture.gotHeader)
	}
}

func TestTraceMiddleware_NoHeader(t *testing.T) {
	t.Parallel()

	var capture captureHeaderHandler

	traced := sloggcp.TraceMiddleware(&capture)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	traced.ServeHTTP(httptest.NewRecorder(), req)

	if capture.gotHeader != "" {
		t.Errorf("expected empty header, got %q", capture.gotHeader)
	}
}

// --- InitCloudRun ---.

func TestInitCloudRun_ReturnsHandler(t *testing.T) {
	t.Parallel()

	h := sloggcp.InitCloudRun()
	if h == nil {
		t.Fatal("InitCloudRun returned nil")
	}

	// Should implement slog.Handler.
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("handler should be enabled at INFO level")
	}
}

// --- Concurrent writes ---.

func TestHandler_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("t1", "s1", false), "proj"))

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(n int) {
			defer wg.Done()

			logger.InfoContext(context.Background(), "concurrent", "n", n) //nolint:sloglint // Test format.
		}(i)
	}

	wg.Wait()

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, goroutines)

	// Verify all event_ids are unique.
	seen := make(map[string]bool, goroutines)

	for _, entry := range entries {
		id, ok := entry["event_id"].(string)
		if !ok {
			t.Error("event_id missing in concurrent entry")

			continue
		}

		if seen[id] {
			t.Errorf("duplicate event_id: %s", id)
		}

		seen[id] = true
	}
}

// --- slogtest compliance ---.

func TestHandler_SlogTestCompliance(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}

	newHandler := func(t *testing.T) slog.Handler {
		t.Helper()

		buf.Reset()

		return sloggcp.NewHandler(
			slog.NewJSONHandler(buf, nil),
			nil,
			"test-project",
			sloggcp.WithEventID(false),
		)
	}

	result := func(t *testing.T) map[string]any {
		t.Helper()

		entries := sloggcptest.LogEntries(buf)
		if len(entries) == 0 {
			return nil
		}

		return entries[len(entries)-1]
	}

	// slogtest.Run tests the handler against the slog.Handler contract.
	slogtest.Run(t, newHandler, result)
}

// --- ErrorAttrs ---.

func TestErrorAttrs(t *testing.T) {
	testErr := errors.New("something failed")
	attrs := sloggcp.ErrorAttrs(testErr, sloggcp.ServiceContext{Service: "my-service", Version: "my-service-00001"})

	if len(attrs) != 4 {
		t.Fatalf("got %d attrs, want 4", len(attrs))
	}

	if attrs[0].Key != "@type" {
		t.Errorf("first attr key = %q, want @type", attrs[0].Key)
	}

	if !strings.Contains(attrs[0].Value.String(), "ReportedErrorEvent") {
		t.Error("@type should reference ReportedErrorEvent")
	}
}

func TestErrorAttrs_NilError(t *testing.T) {
	t.Parallel()

	attrs := sloggcp.ErrorAttrs(nil, sloggcp.ServiceContext{})

	// Should have 3 attrs: @type, serviceContext, stack_trace (no error attr).
	if len(attrs) != 3 {
		t.Fatalf("got %d attrs, want 3", len(attrs))
	}
}

func TestErrorAttrsAny(t *testing.T) {
	t.Parallel()

	testErr := errors.New("something failed")
	anyAttrs := sloggcp.ErrorAttrsAny(testErr, sloggcp.ServiceContext{Service: "my-service", Version: "my-service-00001"})

	// Alternating key-value: "@type", value, slog.Group(...), "stack_trace", trace, "error", errorMsg.
	if len(anyAttrs) != 7 {
		t.Fatalf("got %d items, want 7", len(anyAttrs))
	}

	// First should be the @type key.
	if key, ok := anyAttrs[0].(string); !ok || key != "@type" {
		t.Errorf("first item = %v, want @type string", anyAttrs[0])
	}

	// Second should be the @type value.
	if val, ok := anyAttrs[1].(string); !ok || !strings.Contains(val, "ReportedErrorEvent") {
		t.Errorf("second item = %v, want ReportedErrorEvent type", anyAttrs[1])
	}

	// Error key-value pair (after stack_trace key-value pair).
	if key, ok := anyAttrs[5].(string); !ok || key != "error" {
		t.Errorf("sixth item = %v, want error key", anyAttrs[5])
	}
}

func TestErrorAttrsAny_NilError(t *testing.T) {
	t.Parallel()

	anyAttrs := sloggcp.ErrorAttrsAny(nil, sloggcp.ServiceContext{})

	// Without error: "@type", value, slog.Group(...), "stack_trace", trace.
	if len(anyAttrs) != 5 {
		t.Fatalf("got %d items, want 5", len(anyAttrs))
	}
}

// --- detectProjectID ---.

func TestDetectProjectID_GCPProjectID(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "my-project")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "should-not-use")

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.InfoContext(context.Background(), "detect test")

	entries := sloggcptest.LogEntries(buf)

	got, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.HasPrefix(got, "projects/my-project/") {
		t.Errorf("trace = %q, want prefix projects/my-project/", got)
	}
}

func TestDetectProjectID_Fallback(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("PROJECT_ID", "")

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.InfoContext(context.Background(), "fallback test")

	entries := sloggcptest.LogEntries(buf)

	got, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.HasPrefix(got, "projects/unknown-project/") {
		t.Errorf("trace = %q, want prefix projects/unknown-project/", got)
	}
}

func TestDetectProjectID_BypassMetadataEnv(t *testing.T) {
	t.Setenv("GCP_METADATA_DISABLED", "true")
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("PROJECT_ID", "")

	sloggcp.ResetMetadataCacheForTest()
	defer sloggcp.ResetMetadataCacheForTest()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.InfoContext(context.Background(), "bypass metadata test")

	entries := sloggcptest.LogEntries(buf)

	got, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.HasPrefix(got, "projects/unknown-project/") {
		t.Errorf("trace = %q, want prefix projects/unknown-project/", got)
	}
}

// --- queryMetadataProjectID ---.

func TestDetectProjectID_MetadataFallback(t *testing.T) {
	// Start a fake metadata server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte("metadata-project-id"))
	}))
	defer server.Close()

	// Clear env vars so metadata is the only source.
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("PROJECT_ID", "")

	// Point at our test server and reset cache.
	sloggcp.SetMetadataURLForTest(server.URL)
	sloggcp.ResetMetadataCacheForTest()
	defer func() {
		sloggcp.SetMetadataURLForTest("http://metadata.google.internal/computeMetadata/v1/project/project-id")
		sloggcp.ResetMetadataCacheForTest()
	}()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.Info("test")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	trace, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.Contains(trace, "metadata-project-id") {
		t.Errorf("trace = %q, want to contain metadata-project-id", trace)
	}
}

func TestDetectProjectID_MetadataUnavailable(t *testing.T) {
	// Start a server that always fails.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Clear env vars.
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("PROJECT_ID", "")

	sloggcp.SetMetadataURLForTest(server.URL)
	sloggcp.ResetMetadataCacheForTest()
	defer func() {
		sloggcp.SetMetadataURLForTest("http://metadata.google.internal/computeMetadata/v1/project/project-id")
		sloggcp.ResetMetadataCacheForTest()
	}()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.Info("test")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	trace, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.Contains(trace, "unknown-project") {
		t.Errorf("trace = %q, want to contain unknown-project", trace)
	}
}

func TestDetectProjectID_EnvTakesPrecedenceOverMetadata(t *testing.T) {
	// Start a metadata server that returns a different project.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("metadata-project"))
	}))
	defer server.Close()

	// Set env var — should take precedence.
	t.Setenv("GCP_PROJECT_ID", "env-project")

	sloggcp.SetMetadataURLForTest(server.URL)
	sloggcp.ResetMetadataCacheForTest()
	defer func() {
		sloggcp.SetMetadataURLForTest("http://metadata.google.internal/computeMetadata/v1/project/project-id")
		sloggcp.ResetMetadataCacheForTest()
	}()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, testResolver("trace-1", "span-1", false), ""))

	logger.Info("test")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	trace, _ := entries[0]["logging.googleapis.com/trace"].(string)
	if !strings.Contains(trace, "env-project") {
		t.Errorf("trace = %q, want to contain env-project", trace)
	}
}

// --- Full chain integration ---.

func TestFullChain_CloudLoggingJSON(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}

	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})

	resolver := testResolver("abc123def456", testSpanID, true)
	h := sloggcp.NewHandler(inner, resolver, "my-gcp-project")
	logger := slog.New(h)

	logger.WarnContext(context.Background(), "sync failed")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	entry := entries[0]

	// Check GCP field names.
	if entry["message"] != "sync failed" {
		t.Errorf("message = %v", entry["message"])
	}

	if entry["severity"] != "WARNING" {
		t.Errorf("severity = %v, want WARNING", entry["severity"])
	}

	if _, ok := entry["event_id"].(string); !ok {
		t.Error("event_id missing")
	}

	wantTrace := "projects/my-gcp-project/traces/abc123def456"
	if entry["logging.googleapis.com/trace"] != wantTrace {
		t.Errorf("trace = %v, want %v", entry["logging.googleapis.com/trace"], wantTrace)
	}

	if entry["logging.googleapis.com/spanId"] != testSpanID {
		t.Errorf("spanId = %v", entry["logging.googleapis.com/spanId"])
	}

	if entry["logging.googleapis.com/trace_sampled"] != true {
		t.Errorf("trace_sampled = %v, want true", entry["logging.googleapis.com/trace_sampled"])
	}
}

// --- WithTrace ---.

func TestWithTrace_InjectsTraceID(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	resolver := func(ctx context.Context) sloggcp.TraceContext {
		info := sloggcp.ParseCloudTraceHeaderForTest(sloggcp.TraceHeaderKeyForTest(ctx))
		return sloggcp.TraceContext(info)
	}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, resolver, "test-project"))

	ctx := sloggcp.WithTrace(context.Background())
	logger.InfoContext(ctx, "job started")

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	traceField, ok := entries[0]["logging.googleapis.com/trace"].(string)
	if !ok || traceField == "" {
		t.Error("WithTrace should inject trace ID")
	}

	if !strings.Contains(traceField, "projects/test-project/traces/") {
		t.Errorf("trace = %q, want projects/test-project/traces/...", traceField)
	}
}

// --- HTTPRequestAttr ---.

func TestHTTPRequestAttr_FullFields(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, nil, "test-project", sloggcp.WithEventID(false)))

	reqAttr := sloggcp.HTTPRequestAttr(&sloggcp.HTTPRequest{
		Method:       "GET",
		URL:          testHealthURL,
		Status:       200,
		Latency:      150 * time.Millisecond,
		RemoteIP:     "10.0.0.1",
		UserAgent:    "curl/8.0",
		RequestSize:  0,
		ResponseSize: 42,
	})

	logger.LogAttrs(context.Background(), slog.LevelInfo, "request served", reqAttr)

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	httpReq, ok := entries[0]["httpRequest"].(map[string]any)
	if !ok {
		t.Fatal("httpRequest field missing or not a map")
	}

	if httpReq["requestMethod"] != "GET" {
		t.Errorf("requestMethod = %v, want GET", httpReq["requestMethod"])
	}

	if httpReq["requestUrl"] != testHealthURL {
		t.Errorf("requestUrl = %v, want %s", httpReq["requestUrl"], testHealthURL)
	}

	// Status comes back as float64 from JSON.
	if status, ok := httpReq["status"].(float64); !ok || int(status) != 200 {
		t.Errorf("status = %v, want 200", httpReq["status"])
	}
}

func TestHTTPRequestAttr_MinimalFields(t *testing.T) {
	t.Parallel()

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, nil)
	logger := slog.New(sloggcp.NewHandler(inner, nil, "test-project", sloggcp.WithEventID(false)))

	reqAttr := sloggcp.HTTPRequestAttr(&sloggcp.HTTPRequest{
		Method: "POST",
		URL:    "/api/data",
		Status: 201,
	})

	logger.LogAttrs(context.Background(), slog.LevelInfo, "created", reqAttr)

	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	httpReq, ok := entries[0]["httpRequest"].(map[string]any)
	if !ok {
		t.Fatal("httpRequest field missing")
	}

	// Optional fields should be absent when zero.
	if _, ok := httpReq["remoteIp"]; ok {
		t.Error("remoteIp should be omitted when empty")
	}

	if _, ok := httpReq["userAgent"]; ok {
		t.Error("userAgent should be omitted when empty")
	}
}

func TestHTTPRequestAttr_Nil(t *testing.T) {
	t.Parallel()

	attr := sloggcp.HTTPRequestAttr(nil)
	if attr.Key != "" {
		t.Errorf("expected empty attr key, got %q", attr.Key)
	}
}

// --- LevelVar ---.

func TestWithLevelVar_DynamicLevelChange(t *testing.T) {
	t.Parallel()

	var level slog.LevelVar
	level.Set(slog.LevelInfo)

	buf := &sloggcptest.SyncBuffer{}
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       &level,
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})
	logger := slog.New(sloggcp.NewHandler(inner, nil, "test-project", sloggcp.WithEventID(false)))

	// Should log at INFO.
	logger.Info("visible")
	entries := sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1)

	// Change to ERROR — INFO should now be filtered.
	level.Set(slog.LevelError)
	logger.Info("hidden")
	entries = sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 1) // Still 1.

	// ERROR should still log.
	logger.Error("still visible")
	entries = sloggcptest.LogEntries(buf)
	sloggcptest.AssertLogCount(t, entries, 2) // Now 2.
}

// --- Fuzz tests ---.

func FuzzParseCloudTraceHeader(f *testing.F) {
	f.Add("abc/def;o=1")
	f.Add("")
	f.Add("/;o=0")
	f.Add("trace123/456;o=1")
	f.Add("nospan")

	f.Fuzz(func(t *testing.T, header string) {
		info := sloggcp.ParseCloudTraceHeaderForTest(header)
		_ = info.TraceID
		_ = info.SpanID
		_ = info.Sampled
	})
}

func FuzzHandle(f *testing.F) {
	f.Add("hello world")
	f.Add("")
	f.Add(strings.Repeat("x", 10000))
	f.Add("special chars: \n\t\r\x00")

	f.Fuzz(func(t *testing.T, msg string) {
		buf := &sloggcptest.SyncBuffer{}
		inner := slog.NewJSONHandler(buf, nil)
		h := sloggcp.NewHandler(inner, nil, "fuzz-project")
		logger := slog.New(h)

		logger.Info(msg)

		raw := buf.Bytes()
		if len(raw) == 0 {
			t.Fatal("no output")
		}

		var entry map[string]any
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
	})
}
