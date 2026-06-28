// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"context"
	"sync"
)

// TraceHeaderKeyType is an exported alias of traceHeaderKey for
// use in external tests. This file is only compiled during testing.
type TraceHeaderKeyType = traceHeaderKey

// TraceHeaderKeyForTest extracts the trace header value from context for testing.
func TraceHeaderKeyForTest(ctx context.Context) string {
	v, _ := ctx.Value(traceHeaderKey{}).(string)
	return v
}

// CloudTraceForTest holds parsed trace context, exported for tests.
type CloudTraceForTest struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// ParseCloudTraceHeaderForTest wraps the unexported parseCloudTraceHeader
// for use in external tests.
func ParseCloudTraceHeaderForTest(header string) CloudTraceForTest {
	info := parseCloudTraceHeader(header)

	return CloudTraceForTest{
		TraceID: info.traceID,
		SpanID:  info.spanID,
		Sampled: info.sampled,
	}
}

// ResetMetadataCacheForTest resets the metadata project ID cache,
// allowing tests to re-trigger metadata detection.
func ResetMetadataCacheForTest() {
	metadataProjectID = ""
	metadataOnce = sync.Once{}
}
