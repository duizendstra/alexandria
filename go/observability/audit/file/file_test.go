// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package file_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/observability/audit"
	"github.com/duizendstra/alexandria/go/observability/audit/file"
)

var testClock = func() time.Time {
	return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
}

// countLines returns the number of newline-terminated JSON entries in path,
// failing the test on any line that does not parse as an audit entry.
func countLines(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	count := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e audit.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("torn or malformed line in %s: %q: %v", path, line, err)
		}
		count++
	}

	return count
}

// TestReadScorecard_SkipsMalformedLine pins the torn-write recovery contract:
// a malformed line (e.g. a partial append from a crash) is skipped and every
// entry after it is still counted. The json.Decoder-based implementation this
// replaced looped forever here, because a Decoder cannot resync after a
// syntax error.
func TestReadScorecard_SkipsMalformedLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "audit.log")
	content := `{"ts":"2026-03-01T12:00:00Z","actor":"alice","action":"create","resource":"doc/1"}
{"ts":"2026-03-01T12:00:0
{"ts":"2026-03-01T12:00:02Z","actor":"bob","action":"delete","resource":"doc/2"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	var (
		sc     audit.Scorecard
		scErr  error
		once   sync.Once
		finish = func() { once.Do(func() { close(done) }) }
	)

	go func() {
		sc, scErr = file.ReadScorecard(path)
		finish()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ReadScorecard did not return within 10s — malformed line caused a hang")
	}

	if scErr != nil {
		t.Fatalf("ReadScorecard: %v", scErr)
	}
	if sc.Total != 2 {
		t.Fatalf("Total = %d, want 2 (malformed middle line skipped, trailing entry counted)", sc.Total)
	}
	if sc.ByActor["alice"] != 1 || sc.ByActor["bob"] != 1 {
		t.Fatalf("ByActor = %v, want alice and bob counted once each", sc.ByActor)
	}
}

// TestReadScorecard_UnterminatedFinalLine verifies an entry without a trailing
// newline (append interrupted before the '\n') is still parsed if valid JSON.
func TestReadScorecard_UnterminatedFinalLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "audit.log")
	content := `{"ts":"2026-03-01T12:00:00Z","actor":"alice","action":"create","resource":"doc/1"}` + "\n" +
		`{"ts":"2026-03-01T12:00:01Z","actor":"bob","action":"view","resource":"doc/1"}` // No trailing newline.
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	sc, err := file.ReadScorecard(path)
	if err != nil {
		t.Fatalf("ReadScorecard: %v", err)
	}
	if sc.Total != 2 {
		t.Fatalf("Total = %d, want 2", sc.Total)
	}
}

// TestReadScorecard_TopResourcesOrder pins the "key (count)" format and the
// descending order of the top-resources summary.
func TestReadScorecard_TopResourcesOrder(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "audit.log")

	var b strings.Builder
	writeN := func(resource string, n int) {
		for i := range n {
			fmt.Fprintf(&b, `{"ts":"","actor":"a%d","action":"touch","resource":"%s"}`+"\n", i, resource)
		}
	}
	// Seven distinct resources with distinct counts: only the top five report.
	for i, n := range []int{7, 6, 5, 4, 3, 2, 1} {
		writeN(fmt.Sprintf("res/%d", i), n)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	sc, err := file.ReadScorecard(path)
	if err != nil {
		t.Fatalf("ReadScorecard: %v", err)
	}

	want := []string{"res/0 (7)", "res/1 (6)", "res/2 (5)", "res/3 (4)", "res/4 (3)"}
	if len(sc.TopResources) != len(want) {
		t.Fatalf("TopResources = %v, want %v", sc.TopResources, want)
	}
	for i, w := range want {
		if sc.TopResources[i] != w {
			t.Fatalf("TopResources[%d] = %q, want %q (full: %v)", i, sc.TopResources[i], w, sc.TopResources)
		}
	}
}

// TestFileWriter_RotationKeepsSingleBackup verifies that rotation renames the
// live log to <path>.1, that a second rotation replaces the previous backup,
// and that no line in either file is ever torn.
func TestFileWriter_RotationKeepsSingleBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Each marshaled entry is ~80 bytes; rotate after roughly two entries.
	w, err := file.NewFileWriter(path, file.WithClock(testClock), file.WithMaxLogSize(150))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	for i := range 10 {
		entry := audit.Entry{Actor: "actor", Action: "act", Resource: fmt.Sprintf("res/%d", i)}
		if err := w.Log(context.Background(), entry); err != nil {
			t.Fatalf("Log %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected backup file after rotation: %v", err)
	}
	// Only the single .1 backup is kept — never a .2.
	if _, err := os.Stat(path + ".2"); !os.IsNotExist(err) {
		t.Fatalf("unexpected second backup file (err=%v)", err)
	}

	// Both files contain only whole, parseable lines.
	live := countLines(t, path)
	backup := countLines(t, path+".1")
	if live == 0 || backup == 0 {
		t.Fatalf("expected entries in both live (%d) and backup (%d) logs", live, backup)
	}
}

// TestFileWriter_RotateRenameFailure pins the self-healing contract when the
// backup rename fails: Log returns an error and the failing entry is NOT
// written, but the writer reopens the original file so later appends succeed.
func TestFileWriter_RotateRenameFailure(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(testClock), file.WithMaxLogSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// First entry: size 0 < 1, no rotation; brings written past the limit.
	if err := w.Log(context.Background(), audit.Entry{Actor: "a", Action: "x", Resource: "r/0"}); err != nil {
		t.Fatalf("first Log: %v", err)
	}

	// Make the rename fail: the directory becomes read-only, so the log
	// cannot be renamed to audit.log.1 (directory mutation required).
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err = w.Log(context.Background(), audit.Entry{Actor: "a", Action: "x", Resource: "r/1"})
	if err == nil {
		t.Fatal("Log during failed rotation: want error, got nil")
	}
	if !strings.Contains(err.Error(), "rotate audit log") {
		t.Fatalf("Log error = %v, want rotation failure", err)
	}

	// Restore the directory; the writer must have self-healed onto the
	// original file, and the next rotation attempt now succeeds.
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := w.Log(context.Background(), audit.Entry{Actor: "a", Action: "x", Resource: "r/2"}); err != nil {
		t.Fatalf("Log after directory restored: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// The r/1 entry was dropped (Log returned an error); r/0 rotated into
	// the backup and r/2 is in the fresh live file.
	total := countLines(t, path) + countLines(t, path+".1")
	if total != 2 {
		t.Fatalf("entries across live+backup = %d, want 2 (r/0 and r/2; failed r/1 dropped)", total)
	}
}

// TestFileWriter_ConcurrentLogging hammers one writer from many goroutines
// with rotation enabled; run under -race. Every surviving line must be whole.
func TestFileWriter_ConcurrentLogging(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(testClock), file.WithMaxLogSize(4096))
	if err != nil {
		t.Fatal(err)
	}

	const (
		goroutines = 8
		perG       = 200
	)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range perG {
				entry := audit.Entry{
					Actor:    fmt.Sprintf("actor-%d", g),
					Action:   "write",
					Resource: fmt.Sprintf("res/%d", i),
				}
				if err := w.Log(context.Background(), entry); err != nil {
					t.Errorf("Log: %v", err)

					return
				}
			}
		})
	}
	wg.Wait()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Rotation keeps a single backup, so entries may be discarded with old
	// backups — but every line still present must be whole and parseable,
	// and the live file must hold at most one rotation window.
	live := countLines(t, path)
	if _, err := os.Stat(path + ".1"); err == nil {
		countLines(t, path+".1")
	}
	if live == 0 {
		t.Fatal("no entries in live log after concurrent writes")
	}
}

// TestFileWriter_CloseThenLogFails pins that a closed writer reports errors
// rather than silently dropping entries.
func TestFileWriter_CloseThenLogFails(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "audit.log")
	w, err := file.NewFileWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Log(context.Background(), audit.Entry{Actor: "a", Action: "x", Resource: "r"}); err == nil {
		t.Fatal("Log after Close: want error, got nil")
	}
}
