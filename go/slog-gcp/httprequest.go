// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"fmt"
	"log/slog"
	"net/http"
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
//	    sloggcp.HTTPRequestAttr(&sloggcp.HTTPRequest{
//	        Method:  r.Method,
//	        URL:     r.URL.String(),
//	        Status:  statusCode,
//	        Latency: duration,
//	    }),
//	)
func HTTPRequestAttr(req *HTTPRequest) slog.Attr {
	if req == nil {
		return slog.Attr{}
	}
	attrs := []any{
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		slog.String("requestMethod", req.Method),
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		slog.String("requestUrl", req.URL),
		slog.Int("status", req.Status),
		slog.String("latency", fmt.Sprintf("%fs", req.Latency.Seconds())),
	}

	if req.RemoteIP != "" {
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		attrs = append(attrs, slog.String("remoteIp", req.RemoteIP))
	}

	if req.UserAgent != "" {
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		attrs = append(attrs, slog.String("userAgent", req.UserAgent))
	}

	if req.RequestSize > 0 {
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		attrs = append(attrs, slog.Int64("requestSize", req.RequestSize))
	}

	if req.ResponseSize > 0 {
		//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase keys.
		attrs = append(attrs, slog.Int64("responseSize", req.ResponseSize))
	}

	//nolint:sloglint // GCP Cloud Logging HTTP request payload requires camelCase group name.
	return slog.Group("httpRequest", attrs...)
}

// responseRecorder is a minimal wrapper to capture status and size.
type responseRecorder struct {
	http.ResponseWriter
	Status int
	Size   int64
}

// WriteHeader captures the status code before delegating.
func (r *responseRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write captures the response size and delegates.
func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.Status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	size, err := r.ResponseWriter.Write(b)
	r.Size += int64(size)

	return size, err //nolint:wrapcheck // implementing http.ResponseWriter, must return the unwrapped error.
}

// RequestLoggerMiddleware returns an http.Handler that logs each request
// using sloggcp.HTTPRequestAttr. It intercepts the response to capture
// the status code and response size, dynamically forwarding http.Flusher
// and http.Hijacker interfaces if supported by the underlying ResponseWriter.
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, Status: http.StatusOK}

		var wrapped http.ResponseWriter = rec
		f, hasFlusher := w.(http.Flusher)
		h, hasHijacker := w.(http.Hijacker)

		switch {
		case hasFlusher && hasHijacker:
			wrapped = &flushHijackRecorder{responseRecorder: rec, Flusher: f, Hijacker: h}
		case hasFlusher:
			wrapped = &flushRecorder{responseRecorder: rec, Flusher: f}
		case hasHijacker:
			wrapped = &hijackRecorder{responseRecorder: rec, Hijacker: h}
		}

		next.ServeHTTP(wrapped, r)

		latency := time.Since(start)

		reqAttr := HTTPRequest{
			Method:       r.Method,
			URL:          r.URL.String(),
			Status:       rec.Status,
			UserAgent:    r.UserAgent(),
			RemoteIP:     r.RemoteAddr,
			Latency:      latency,
			ResponseSize: rec.Size,
			RequestSize:  r.ContentLength,
		}

		slog.LogAttrs(r.Context(), slog.LevelInfo, "HTTP Request", HTTPRequestAttr(&reqAttr))
	})
}

type flushRecorder struct {
	*responseRecorder
	http.Flusher
}

type hijackRecorder struct {
	*responseRecorder
	http.Hijacker
}

type flushHijackRecorder struct {
	*responseRecorder
	http.Flusher
	http.Hijacker
}


