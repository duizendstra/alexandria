// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"fmt"
	"log/slog"
	"time"
)

// HTTPRequest captures HTTP request and response metadata for the
// Cloud Logging httpRequest special field.
//
// See: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
type HTTPRequest struct {
	Method       string
	URL          string
	Status       int
	Latency      time.Duration
	RemoteIP     string
	UserAgent    string
	RequestSize  int64
	ResponseSize int64
}

// HTTPRequestAttr returns a slog.Attr formatted as the Cloud Logging
// httpRequest special field. Cloud Logging natively renders this in
// the UI with method, URL, status, latency, and more.
//
//	slog.LogAttrs(ctx, slog.LevelInfo, "request served",
//	    sloggcp.HTTPRequestAttr(sloggcp.HTTPRequest{
//	        Method:  r.Method,
//	        URL:     r.URL.String(),
//	        Status:  statusCode,
//	        Latency: duration,
//	    }),
//	)
func HTTPRequestAttr(req HTTPRequest) slog.Attr {
	attrs := []any{
		slog.String("requestMethod", req.Method),
		slog.String("requestUrl", req.URL),
		slog.Int("status", req.Status),
		slog.String("latency", fmt.Sprintf("%fs", req.Latency.Seconds())),
	}

	if req.RemoteIP != "" {
		attrs = append(attrs, slog.String("remoteIp", req.RemoteIP))
	}

	if req.UserAgent != "" {
		attrs = append(attrs, slog.String("userAgent", req.UserAgent))
	}

	if req.RequestSize > 0 {
		attrs = append(attrs, slog.Int64("requestSize", req.RequestSize))
	}

	if req.ResponseSize > 0 {
		attrs = append(attrs, slog.Int64("responseSize", req.ResponseSize))
	}

	return slog.Group("httpRequest", attrs...)
}
