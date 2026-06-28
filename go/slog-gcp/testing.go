// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"
)

// SyncBuffer is a thread-safe bytes.Buffer for capturing log output
// in concurrent tests.
type SyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer.
func (b *SyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// String returns the accumulated output.
func (b *SyncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// Reset clears the buffer.
func (b *SyncBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf.Reset()
}

// Bytes returns a copy of the accumulated output.
func (b *SyncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return bytes.Clone(b.buf.Bytes())
}

// NewTestLogger creates a slog.Logger backed by a SyncBuffer for use
// in tests. Returns the logger and the buffer for assertion helpers.
func NewTestLogger(t *testing.T, opts ...Option) (*slog.Logger, *SyncBuffer) {
	t.Helper()

	buf := &SyncBuffer{}
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: GCPReplaceAttr,
	})
	h := NewHandler(inner, nil, "test-project", opts...)

	return slog.New(h), buf
}

// LogEntries parses newline-delimited JSON log output into a slice of
// maps. Useful for asserting on structured log fields in tests.
func LogEntries(buf *SyncBuffer) []map[string]any {
	raw := buf.Bytes()
	lines := bytes.Split(raw, []byte("\n"))

	var entries []map[string]any

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries
}

// AssertLogContains checks that at least one log entry contains the
// given key with the given value.
func AssertLogContains(t *testing.T, entries []map[string]any, key string, value any) {
	t.Helper()

	for _, entry := range entries {
		if v, ok := entry[key]; ok && v == value {
			return
		}
	}

	t.Errorf("no log entry contains %s=%v", key, value)
}

// AssertLogLevel checks the severity of the log entry at the given index.
func AssertLogLevel(t *testing.T, entries []map[string]any, index int, level string) {
	t.Helper()

	if index >= len(entries) {
		t.Fatalf("entry index %d out of range (have %d entries)", index, len(entries))
	}

	got, _ := entries[index]["severity"].(string)
	if got != level {
		t.Errorf("entry[%d] severity = %q, want %q", index, got, level)
	}
}

// AssertLogCount checks the number of log entries.
func AssertLogCount(t *testing.T, entries []map[string]any, expected int) {
	t.Helper()

	if len(entries) != expected {
		t.Errorf("got %d log entries, want %d", len(entries), expected)
	}
}
