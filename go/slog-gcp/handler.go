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
	pid := pidStr()

	var arr [64]byte
	buf := arr[:0]
	buf = append(buf, pid...)
	buf = append(buf, '-')
	buf = strconv.AppendUint(buf, val, 10) //nolint:mnd // Base 10 is standard.

	return string(buf)
}

// IDResolver extracts trace and span IDs from context.
// The log package does not know how traces are stored — callers provide
// the bridge via this function type.
type IDResolver func(ctx context.Context) TraceContext

// handler wraps an inner slog.Handler and auto-injects insertId and
// GCP Cloud Logging trace fields into every log record.
//
// Cloud Logging only honours the reserved logging.googleapis.com/* keys at
// the top level of the entry, so they must never be nested under an active
// WithGroup. The handler therefore keeps root — the inner handler as it was
// before the first WithGroup — and replays the group/attr chain recorded in
// steps on top of root.WithAttrs(reserved) for each record written while a
// group is active. The no-group path is untouched: reserved attrs are
// appended to the record and inner handles it directly.
type handler struct {
	inner       slog.Handler
	root        slog.Handler
	steps       []chainStep
	resolve     IDResolver
	projectID   string
	tracePrefix string
	insertID    bool
	insertIDKey string
}

// chainStep is one WithGroup or WithAttrs call made after the first group
// was opened, in call order. A step is a group when group is non-empty,
// otherwise it carries attrs.
type chainStep struct {
	group string
	attrs []slog.Attr
}

// Option configures [NewHandler].
type Option func(*handler)

// WithInsertID controls whether a unique insertId is generated per log
// record for Cloud Logging deduplication. Default is false.
func WithInsertID(enabled bool) Option {
	return func(h *handler) {
		h.insertID = enabled
	}
}

// WithInsertIDKey configures a custom field key used for the insert ID.
// If not specified or empty, it defaults to [FieldInsertID] ("logging.googleapis.com/insertId").
func WithInsertIDKey(key string) Option {
	return func(h *handler) {
		if key != "" {
			h.insertIDKey = key
		}
	}
}

// WithEventID is a backward-compatible alias for [WithInsertID].
// By default it emits under [FieldInsertID] ("logging.googleapis.com/insertId").
// To preserve legacy "event_id" field names, combine with WithInsertIDKey("event_id").
func WithEventID(enabled bool) Option {
	return WithInsertID(enabled)
}

// NewHandler wraps an inner [slog.Handler] and auto-injects insertId
// and GCP Cloud Logging trace fields into every log record.
//
// The resolve function extracts trace and span IDs from context.
// Pass nil to disable trace injection (only insertId is added).
// If projectID is empty, it is auto-detected from GCP environment variables.
//
//	inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
//	    ReplaceAttr: log.GCPReplaceAttr,
//	})
//	slog.SetDefault(slog.New(log.NewHandler(inner, resolver, "")))
func NewHandler(inner slog.Handler, resolve IDResolver, projectID string, opts ...Option) slog.Handler {
	if inner == nil {
		panic("sloggcp: inner handler cannot be nil")
	}

	if projectID == "" {
		projectID = detectProjectID()
	}

	h := &handler{
		inner:       inner,
		root:        inner,
		steps:       nil,
		resolve:     resolve,
		projectID:   projectID,
		tracePrefix: "projects/" + projectID + "/traces/",
		insertID:    false,
		insertIDKey: FieldInsertID,
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

// Handle injects insertId and GCP trace fields, then delegates to the
// inner handler. The resolver is called once per log line.
//
// The reserved fields always land at the payload root: without an active
// group they are appended to the record; with one, they are applied to the
// ungrouped root handler and the group chain is replayed on top (#256).
func (h *handler) Handle(ctx context.Context, rec slog.Record) error { //nolint:gocritic // slog.Record passed by value per slog.Handler contract.
	reserved := h.reservedAttrs(ctx)

	if len(reserved) == 0 || len(h.steps) == 0 {
		rec.AddAttrs(reserved...)

		return h.inner.Handle(ctx, rec) //nolint:wrapcheck // Error context sufficient from caller.
	}

	target := h.root.WithAttrs(reserved)

	for _, step := range h.steps {
		if step.group != "" {
			target = target.WithGroup(step.group)
		} else {
			target = target.WithAttrs(step.attrs)
		}
	}

	return target.Handle(ctx, rec) //nolint:wrapcheck // Error context sufficient from caller.
}

// WithAttrs returns a new handler with the given attributes.
//
// Before the first group the attrs also extend root, so they stay at the
// payload root on the replay path; after it they are recorded as a step.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	next.inner = h.inner.WithAttrs(attrs)

	if len(h.steps) == 0 {
		next.root = h.root.WithAttrs(attrs)
	} else {
		next.steps = append(next.steps, chainStep{group: "", attrs: attrs})
	}

	return next
}

// WithGroup returns a new handler with the given group name. An empty
// name returns the receiver, per the slog.Handler contract.
func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	next := h.clone()
	next.inner = h.inner.WithGroup(name)
	next.steps = append(next.steps, chainStep{group: name, attrs: nil})

	return next
}

// reservedAttrs builds the Cloud Logging reserved attrs for one record:
// the insertId when enabled, and the trace fields when the resolver finds
// a trace in ctx. It returns nil when there is nothing to inject.
func (h *handler) reservedAttrs(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	if h.insertID {
		attrs = append(attrs, slog.String(h.insertIDKey, fastUniqueID()))
	}

	if h.resolve != nil {
		tc := h.resolve(ctx)

		if !tc.IsEmpty() {
			attrs = append(attrs,
				slog.String(FieldTrace, h.tracePrefix+tc.TraceID),
				slog.String(FieldSpanID, tc.SpanID),
				slog.Bool(FieldTraceSampled, tc.Sampled),
			)
		}
	}

	return attrs
}

// clone returns a copy of h whose steps slice does not alias h.steps, so
// sibling derived handlers never overwrite each other's chain.
func (h *handler) clone() *handler {
	steps := make([]chainStep, len(h.steps), len(h.steps)+1)
	copy(steps, h.steps)

	return &handler{
		inner:       h.inner,
		root:        h.root,
		steps:       steps,
		resolve:     h.resolve,
		projectID:   h.projectID,
		tracePrefix: h.tracePrefix,
		insertID:    h.insertID,
		insertIDKey: h.insertIDKey,
	}
}
