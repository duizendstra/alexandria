// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// traceHeaderKey is the context key for the X-Cloud-Trace-Context
// header value stored by [TraceMiddleware].
type traceHeaderKey struct{}

// InitCloudRun creates and returns a slog.Handler configured for Cloud
// Run structured logging. On Cloud Run (K_SERVICE set), it outputs
// JSON with GCP Cloud Logging field names. Locally, it uses the
// default text handler.
//
// This function returns the handler for testability. Use [Setup] for
// the common case of calling slog.SetDefault.
func InitCloudRun() slog.Handler {
	level := parseLogLevel()
	format := os.Getenv("LOG_FORMAT")

	// Local development: text handler unless explicitly overridden.
	if os.Getenv("K_SERVICE") == "" && !strings.EqualFold(format, "json") {
		return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	}

	// Cloud Run or explicit JSON: structured JSON with GCP fields.
	if strings.EqualFold(format, "text") {
		return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	}

	resolver := func(ctx context.Context) (string, string, bool) {
		info := parseCloudTraceHeader(traceHeaderFromCtx(ctx))

		return info.traceID, info.spanID, info.sampled
	}

	inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:       level,
		AddSource:   true,
		ReplaceAttr: GCPReplaceAttr,
	})

	return NewHandler(inner, resolver, "")
}

// Setup configures the default slog logger for the current environment.
// It calls [InitCloudRun] and sets the result as the default logger.
func Setup() {
	slog.SetDefault(slog.New(InitCloudRun()))
}

// TraceMiddleware extracts X-Cloud-Trace-Context from HTTP requests
// and stores it in context for downstream slog calls.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-Cloud-Trace-Context")
		if header != "" {
			ctx := context.WithValue(r.Context(), traceHeaderKey{}, header)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// traceHeaderFromCtx retrieves the X-Cloud-Trace-Context header value
// from context, as stored by [TraceMiddleware].
func traceHeaderFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(traceHeaderKey{}).(string)

	return v
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
		if os.Getenv("K_SERVICE") != "" {
			return slog.LevelInfo
		}

		return slog.LevelDebug
	}
}
