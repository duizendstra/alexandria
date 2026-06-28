// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// GCPReplaceAttr maps Go slog field names and values to GCP Cloud
// Logging equivalents: msg → message, level → severity with value
// mapping (DEBUG/INFO/WARNING/ERROR).
func GCPReplaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.MessageKey {
		a.Key = "message"
	}

	if a.Key == slog.LevelKey {
		a.Key = "severity"

		if level, ok := a.Value.Any().(slog.Level); ok {
			switch {
			case level < slog.LevelInfo:
				a.Value = slog.StringValue("DEBUG")
			case level < slog.LevelWarn:
				a.Value = slog.StringValue("INFO")
			case level < slog.LevelError:
				a.Value = slog.StringValue("WARNING")
			default:
				a.Value = slog.StringValue("ERROR")
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
// automatically picks them up.
func ErrorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("@type",
			"type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"),
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}

// ErrorAttrsAny returns error reporting attributes as []any for use
// with slog's alternating key-value API (e.g. slog.Info("msg", attrs...)).
func ErrorAttrsAny() []any {
	return []any{
		"@type",
		"type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
	}
}

// detectProjectID reads the GCP project ID from environment variables.
// Checks GCP_PROJECT_ID (Boozed convention) in addition to the
// standard GCP environment variables.
func detectProjectID() string {
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

	return "unknown-project"
}
