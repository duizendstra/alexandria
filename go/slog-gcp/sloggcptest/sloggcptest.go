// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcptest

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

// SyncBuffer is a thread-safe bytes.Buffer for capturing log output
// in tests.
type SyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer with mutex protection.
func (b *SyncBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// String returns the buffer contents as a string.
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

// Bytes returns a copy of the buffer contents.
func (b *SyncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]byte(nil), b.buf.Bytes()...)
}

// NewTestLogger creates a logger and buffer for testing.
// The logger writes JSON with GCP field names and no event_id.
func NewTestLogger(t *testing.T) (*slog.Logger, *SyncBuffer) {
	t.Helper()

	buf := &SyncBuffer{}
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})

	logger := slog.New(sloggcp.NewHandler(inner, nil, "test-project", sloggcp.WithEventID(false)))

	return logger, buf
}

// LogEntries parses the buffer into a slice of JSON maps.
func LogEntries(buf *SyncBuffer) []map[string]any {
	var entries []map[string]any

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}

		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries
}

// AssertLogContains checks that at least one log entry has the given
// key-value pair.
func AssertLogContains(t *testing.T, entries []map[string]any, key, want string) {
	t.Helper()

	for _, entry := range entries {
		if v, ok := entry[key].(string); ok && v == want {
			return
		}
	}

	t.Errorf("no log entry with %s=%q", key, want)
}

// AssertLogLevel checks that at least one log entry has the given
// severity level.
func AssertLogLevel(t *testing.T, entries []map[string]any, want string) {
	t.Helper()
	AssertLogContains(t, entries, "severity", want)
}

// AssertLogCount checks that the number of log entries matches want.
func AssertLogCount(t *testing.T, entries []map[string]any, want int) {
	t.Helper()

	if len(entries) != want {
		t.Errorf("got %d log entries, want %d", len(entries), want)
	}
}
