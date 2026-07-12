// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
)

// traceContextKey is the context key for the pre-parsed TraceContext struct.
type traceContextKey struct{}

// SetupOption configures [Setup] and [InitCloudRun].
type SetupOption func(*setupConfig)

type setupConfig struct {
	levelVar  *slog.LevelVar
	projectID string
	eventID   *bool
	resolver  IDResolver
	labels    map[string]string
}

// WithLevelVar configures the logger to use the given [slog.LevelVar]
// for dynamic level control. The level can be changed at runtime
// without redeploying the service.
//
//	var level slog.LevelVar
//	sloggcp.Setup(sloggcp.WithLevelVar(&level))
//	// Later, dynamically change:
//	level.Set(slog.LevelError)
func WithLevelVar(lv *slog.LevelVar) SetupOption {
	return func(cfg *setupConfig) {
		cfg.levelVar = lv
	}
}

// WithProjectID configures the logger with an explicit GCP project ID.
// If set, it bypasses the auto-detection metadata server query.
func WithProjectID(id string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.projectID = id
	}
}

// WithEventIDEnabled configures whether a unique event_id is generated per log line.
func WithEventIDEnabled(enabled bool) SetupOption {
	return func(cfg *setupConfig) {
		cfg.eventID = &enabled
	}
}

// WithTraceResolver configures a custom trace ID resolver.
func WithTraceResolver(resolver IDResolver) SetupOption {
	return func(cfg *setupConfig) {
		cfg.resolver = resolver
	}
}

// WithLabels configures a map of global labels injected into all log entries.
func WithLabels(labels map[string]string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.labels = labels
	}
}

// InitCloudRun creates and returns a slog.Handler configured for Cloud
// Run structured logging. On Cloud Run (K_SERVICE set), it outputs
// JSON with GCP Cloud Logging field names. Locally, it uses the
// default text handler.
//
// This function returns the handler for testability. Use [Setup] for
// the common case of calling slog.SetDefault.
func InitCloudRun(opts ...SetupOption) slog.Handler {
	cfg := &setupConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var level slog.Leveler
	if cfg.levelVar != nil {
		level = cfg.levelVar
	} else {
		level = parseLogLevel()
	}
	format := os.Getenv("LOG_FORMAT")

	// If not running on GCP (no K_SERVICE, CLOUD_RUN_JOB, or KUBERNETES_SERVICE_HOST),
	// use a human-readable text handler unless JSON format is explicitly requested.
	isGCP := os.Getenv("K_SERVICE") != "" || os.Getenv("CLOUD_RUN_JOB") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != ""
	if (!isGCP && !strings.EqualFold(format, "json")) || strings.EqualFold(format, "text") {
		return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	}

	var resolver IDResolver
	if cfg.resolver != nil {
		resolver = cfg.resolver
	} else {
		resolver = func(ctx context.Context) TraceContext {
			if ctx == nil {
				return TraceContext{}
			}
			if tc, ok := ctx.Value(traceContextKey{}).(TraceContext); ok {
				return tc
			}

			return TraceContext{}
		}
	}

	inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:       level,
		AddSource:   true,
		ReplaceAttr: GCPReplaceAttr,
	})

	var handlerOpts []Option
	if cfg.eventID != nil {
		handlerOpts = append(handlerOpts, WithEventID(*cfg.eventID))
	}

	h := NewHandler(inner, resolver, cfg.projectID, handlerOpts...)

	if len(cfg.labels) > 0 {
		labelAttrs := make([]any, 0, len(cfg.labels))
		for k, v := range cfg.labels {
			labelAttrs = append(labelAttrs, slog.String(k, v))
		}
		h = h.WithAttrs([]slog.Attr{slog.Group("logging.googleapis.com/labels", labelAttrs...)}) //nolint:sloglint // GCP Logging uses this exact group name.
	}

	return h
}

// Setup configures the default slog logger for the current environment.
// It calls [InitCloudRun] and sets the result as the default logger.
func Setup(opts ...SetupOption) {
	slog.SetDefault(slog.New(InitCloudRun(opts...)))
}

// TraceMiddleware extracts X-Cloud-Trace-Context or traceparent from HTTP requests
// and stores it in context for downstream slog calls.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-Cloud-Trace-Context")
		var tc TraceContext
		if header != "" {
			tc = parseCloudTraceHeader(header)
		} else if tpHeader := r.Header.Get("traceparent"); tpHeader != "" { //nolint:canonicalheader // W3C standard traceparent is always lower-case.
			tc = parseTraceparentHeader(tpHeader)
		}

		if tc.TraceID != "" {
			ctx := context.WithValue(r.Context(), traceContextKey{}, tc)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// WithTrace returns a context with a new trace ID for correlating logs
// in non-HTTP environments (Cloud Run Jobs, workers, Pub/Sub handlers).
// The generated trace context is picked up by the handler's IDResolver
// transparently, just like TraceMiddleware does for HTTP requests.
func WithTrace(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	traceID := fmt.Sprintf("%016x%016x", rand.Uint64(), rand.Uint64()) //nolint:gosec // Weak random ID generation is sufficient for trace IDs.
	tc := TraceContext{
		TraceID: traceID,
		SpanID:  "0000000000000000",
		Sampled: true,
	}

	return context.WithValue(ctx, traceContextKey{}, tc)
}

// WithTraceContext returns a context containing the given TraceContext.
// This allows background workers (e.g. Pub/Sub processors) to propagate
// trace context received from external systems.
func WithTraceContext(ctx context.Context, tc TraceContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, traceContextKey{}, tc)
}

// parseLogLevel returns the slog.Level corresponding to the LOG_LEVEL
// environment variable. Defaults to INFO on Cloud Run, DEBUG locally.
func parseLogLevel() slog.Level {
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		if os.Getenv("K_SERVICE") != "" || os.Getenv("CLOUD_RUN_JOB") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			return slog.LevelInfo
		}

		return slog.LevelDebug
	}
}
