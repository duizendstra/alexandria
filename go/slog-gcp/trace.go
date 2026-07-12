// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	maxTraceIDLen = 32
	maxSpanIDLen  = 16
)

// TraceContext holds parsed trace context from a request.
type TraceContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// IsEmpty reports whether the trace context has no trace ID.
func (tc TraceContext) IsEmpty() bool {
	return tc.TraceID == ""
}

// parseCloudTraceHeader parses the X-Cloud-Trace-Context header.
// Format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
//
// The span ID in the header is decimal; this function converts it to
// 16-character zero-padded hexadecimal as Cloud Logging expects.
func parseCloudTraceHeader(header string) TraceContext {
	var info TraceContext

	traceSpan, params, found := strings.Cut(header, ";")
	if found && strings.Contains(params, "o=1") {
		info.Sampled = true
	}

	traceID, spanDec, found := strings.Cut(traceSpan, "/")
	if len(traceID) > maxTraceIDLen {
		traceID = traceID[:maxTraceIDLen]
	}
	info.TraceID = traceID

	if found {
		spanVal := decimalToHexSpan(spanDec)
		if len(spanVal) > maxSpanIDLen {
			spanVal = spanVal[:maxSpanIDLen]
		}
		info.SpanID = spanVal
	}

	return info
}

func decimalToHexSpan(decimal string) string {
	n, err := strconv.ParseUint(decimal, 10, 64)
	if err != nil {
		// Not a decimal number — return as-is (may already be hex).
		return decimal
	}

	return fmt.Sprintf("%016x", n)
}

// parseTraceparentHeader parses the W3C traceparent header.
// Format: version-trace_id-parent_id-trace_flags.
func parseTraceparentHeader(header string) TraceContext {
	var info TraceContext

	// Skip version.
	_, rest, found := strings.Cut(header, "-")
	if !found {
		return info
	}
	traceID, rest, found := strings.Cut(rest, "-")
	if !found {
		return info
	}
	spanID, flags, found := strings.Cut(rest, "-")
	if !found {
		return info
	}

	// Bounds checking.
	if len(traceID) > maxTraceIDLen {
		traceID = traceID[:maxTraceIDLen]
	}
	if len(spanID) > maxSpanIDLen {
		spanID = spanID[:maxSpanIDLen]
	}

	info.TraceID = traceID
	info.SpanID = spanID
	if flags == "01" {
		info.Sampled = true
	}

	return info
}
