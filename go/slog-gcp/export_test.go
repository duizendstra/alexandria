// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"context"
)

// TraceContextKeyType is an exported alias of traceContextKey for
// use in external tests. This file is only compiled during testing.
type TraceContextKeyType = traceContextKey

// TraceContextForTest extracts the TraceContext struct from context for testing.
func TraceContextForTest(ctx context.Context) TraceContext {
	v, _ := ctx.Value(traceContextKey{}).(TraceContext)

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

	return CloudTraceForTest(info)
}

// ResetMetadataCacheForTest resets the metadata project ID cache,
// allowing tests to re-trigger metadata detection.
func ResetMetadataCacheForTest() {
	metadataMu.Lock()
	defer metadataMu.Unlock()
	metadataProjectID = ""
}

// SetMetadataURLForTest overrides the metadata service URL for testing.
func SetMetadataURLForTest(url string) {
	metadataURL = url
}
