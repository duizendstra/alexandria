// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GCPReplaceAttr maps Go slog field names and values to GCP Cloud
// Logging equivalents: msg → message, time → timestamp,
// source → logging.googleapis.com/sourceLocation, and level → severity
// with value mapping (DEBUG/INFO/NOTICE/WARNING/ERROR/CRITICAL/ALERT/EMERGENCY).
func GCPReplaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.MessageKey {
		a.Key = "message"
	}

	if a.Key == slog.TimeKey {
		a.Key = "timestamp"
	}

	if a.Key == slog.SourceKey {
		a.Key = "logging.googleapis.com/sourceLocation"

		if source, ok := a.Value.Any().(*slog.Source); ok {
			a.Value = slog.GroupValue(
				slog.String("file", source.File),
				slog.Int("line", source.Line),
				slog.String("function", source.Function),
			)
		}
	}

	if a.Key == slog.LevelKey {
		a.Key = "severity"

		if level, ok := a.Value.Any().(slog.Level); ok {
			switch {
			case level < slog.LevelInfo:
				a.Value = slog.StringValue("DEBUG")
			case level < slog.LevelInfo+2: //nolint:mnd // NOTICE threshold.
				a.Value = slog.StringValue("INFO")
			case level < slog.LevelWarn:
				a.Value = slog.StringValue("NOTICE")
			case level < slog.LevelError:
				a.Value = slog.StringValue("WARNING")
			case level < slog.LevelError+4: //nolint:mnd // CRITICAL threshold.
				a.Value = slog.StringValue("ERROR")
			case level < slog.LevelError+8: //nolint:mnd // ALERT threshold.
				a.Value = slog.StringValue("CRITICAL")
			case level < slog.LevelError+12: //nolint:mnd // EMERGENCY threshold.
				a.Value = slog.StringValue("ALERT")
			default:
				a.Value = slog.StringValue("EMERGENCY")
			}
		}
	}

	return a
}

// cloudTrace holds trace context parsed from X-Cloud-Trace-Context.
type cloudTrace struct {
	traceID string
	spanID  string // 16-char hex, converted from decimal.
	sampled bool
}

// parseCloudTraceHeader parses the X-Cloud-Trace-Context header.
// Format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
//
// The span ID in the header is decimal; this function converts it to
// 16-character zero-padded hexadecimal as Cloud Logging expects.
func parseCloudTraceHeader(header string) cloudTrace {
	var info cloudTrace

	parts := strings.SplitN(header, ";", 2) //nolint:mnd // Clear from context.
	traceSpan := parts[0]

	if len(parts) > 1 && strings.Contains(parts[1], "o=1") {
		info.sampled = true
	}

	tsParts := strings.SplitN(traceSpan, "/", 2) //nolint:mnd // Clear from context.
	info.traceID = tsParts[0]

	if len(tsParts) > 1 {
		info.spanID = decimalToHexSpan(tsParts[1])
	}

	return info
}

// decimalToHexSpan converts a decimal span ID string to a 16-character
// zero-padded hexadecimal string. Returns the input unchanged if
// parsing fails.
func decimalToHexSpan(decimal string) string {
	n, err := strconv.ParseUint(decimal, 10, 64)
	if err != nil {
		// Not a decimal number — return as-is (may already be hex).
		return decimal
	}

	return fmt.Sprintf("%016x", n)
}

// ErrorAttrs returns slog attributes for Cloud Error Reporting.
// Attach these to error-level log lines so GCP Error Reporting
// automatically picks them up. Includes a stack trace for grouping.
func ErrorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("@type",
			"type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"),
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
		slog.String("stack_trace", stackTrace(3)),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}

// ErrorAttrsAny returns error reporting attributes as []any for use
// with slog's alternating key-value API (e.g. slog.Error("msg", attrs...)).
func ErrorAttrsAny(err error) []any {
	attrs := []any{
		"@type",
		"type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
		"stack_trace", stackTrace(3),
	}

	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	return attrs
}

var (
	metadataProjectID string
	metadataOnce      sync.Once
)

// detectProjectID reads the GCP project ID from environment variables,
// then falls back to the GCE metadata service on managed GCP platforms.
func detectProjectID() string {
	// Priority 1: Environment variables (fast, overridable).
	for _, key := range []string{
		"GCP_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
		"GCP_PROJECT",
		"PROJECT_ID",
	} {
		if id := os.Getenv(key); id != "" {
			return id
		}
	}

	// Priority 2: GCE metadata service (available on Cloud Run, GKE, etc.).
	metadataOnce.Do(func() {
		metadataProjectID = queryMetadataProjectID()
	})

	if metadataProjectID != "" {
		return metadataProjectID
	}

	return "unknown-project"
}

// queryMetadataProjectID queries the GCE metadata service for the project ID.
// Uses a short timeout to avoid blocking on non-GCP environments.
func queryMetadataProjectID() string {
	const metadataURL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"

	client := &http.Client{Timeout: 500 * time.Millisecond} //nolint:mnd // Short timeout for non-GCP fallback.

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}
