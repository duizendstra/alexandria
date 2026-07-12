// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"strconv"
	"strings"
)

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

func decimalToHexSpan(decimal string) string {
	n, err := strconv.ParseUint(decimal, 10, 64)
	if err != nil {
		// Not a decimal number — return as-is (may already be hex).
		return decimal
	}

	var buf [16]byte
	const hexChars = "0123456789abcdef"
	for i := 15; i >= 0; i-- {
		buf[i] = hexChars[n&0xf]
		n >>= 4
	}
	return string(buf[:])
}
