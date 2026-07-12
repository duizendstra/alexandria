// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

var (
	//nolint:gochecknoglobals // Global sequence counter for generating unique event IDs.
	seq atomic.Uint64

	//nolint:gochecknoglobals // Process-wide cached PID string resolved lazily.
	pidStr = sync.OnceValue(func() string {
		return strconv.Itoa(os.Getpid())
	})
)

func fastUniqueID() string {
	val := seq.Add(1)

	return pidStr() + "-" + strconv.FormatUint(val, 10)
}

// IDResolver extracts trace and span IDs from context.
// The log package does not know how traces are stored — callers provide
// the bridge via this function type.
type IDResolver func(ctx context.Context) TraceContext

// handler wraps an inner slog.Handler and auto-injects event_id and
// GCP Cloud Logging trace fields into every log record.
type handler struct {
	inner       slog.Handler
	resolve     IDResolver
	projectID   string
	tracePrefix string
	eventID     bool
}

// Option configures [NewHandler].
type Option func(*handler)

// WithEventID controls whether a unique event_id is generated per log
// record. Default is false.
func WithEventID(enabled bool) Option {
	return func(h *handler) {
		h.eventID = enabled
	}
}

// NewHandler wraps an inner [slog.Handler] and auto-injects event_id
// and GCP Cloud Logging trace fields into every log record.
//
// The resolve function extracts trace and span IDs from context.
// Pass nil to disable trace injection (only event_id is added).
// If projectID is empty, it is auto-detected from GCP environment variables.
//
//	inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
//	    ReplaceAttr: log.GCPReplaceAttr,
//	})
//	slog.SetDefault(slog.New(log.NewHandler(inner, resolver, "")))
func NewHandler(inner slog.Handler, resolve IDResolver, projectID string, opts ...Option) slog.Handler {
	if projectID == "" {
		projectID = detectProjectID()
	}

	h := &handler{
		inner:       inner,
		resolve:     resolve,
		projectID:   projectID,
		tracePrefix: "projects/" + projectID + "/traces/",
		eventID:     false,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Enabled delegates to the inner handler.
func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle injects event_id and GCP trace fields, then delegates to the
// inner handler. The resolver is called once per log line.
func (h *handler) Handle(ctx context.Context, rec slog.Record) error { //nolint:gocritic // slog.Record passed by value per slog.Handler contract.
	if h.eventID {
		rec.AddAttrs(slog.String("event_id", fastUniqueID()))
	}

	if h.resolve != nil {
		tc := h.resolve(ctx)

		if !tc.IsEmpty() {
			rec.AddAttrs(
				slog.String(FieldTrace, h.tracePrefix+tc.TraceID),
				slog.String(FieldSpanID, tc.SpanID),
				slog.Bool(FieldTraceSampled, tc.Sampled),
			)
		}
	}

	return h.inner.Handle(ctx, rec) //nolint:wrapcheck // Error context sufficient from caller.
}

// WithAttrs returns a new handler with the given attributes.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		inner:       h.inner.WithAttrs(attrs),
		resolve:     h.resolve,
		projectID:   h.projectID,
		tracePrefix: h.tracePrefix,
		eventID:     h.eventID,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{
		inner:       h.inner.WithGroup(name),
		resolve:     h.resolve,
		projectID:   h.projectID,
		tracePrefix: h.tracePrefix,
		eventID:     h.eventID,
	}
}

