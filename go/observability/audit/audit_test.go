package audit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/observability/audit"
	"github.com/duizendstra/alexandria/go/observability/audit/file"
)

const (
	actionCreateIdeas = "ideas.create"
	actionView        = "view"
	actorMCP          = "mcp"
	actorCLI          = "cli"
	resourceIdeasABC  = "ideas/abc123"
)

var fixedTime = func() time.Time {
	return time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
}

// TestEntry_MarshalJSON_GoldenWireFormat pins the exact wire format so the
// time.Time migration cannot drift from what the string-based Entry produced.
func TestEntry_MarshalJSON_GoldenWireFormat(t *testing.T) {
	entry := audit.Entry{
		Time:     fixedTime(),
		Actor:    actorMCP,
		Action:   actionCreateIdeas,
		Resource: resourceIdeasABC,
	}

	got, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"ts":"2026-02-13T14:00:00Z","actor":"mcp","action":"ideas.create","resource":"ideas/abc123"}`
	if string(got) != want {
		t.Errorf("wire format changed:\ngot:  %s\nwant: %s", got, want)
	}
}

// TestEntry_MarshalJSON_ZeroTime pins the historical encoding of an unstamped
// entry: the zero Time is written as an empty "ts" string.
func TestEntry_MarshalJSON_ZeroTime(t *testing.T) {
	got, err := json.Marshal(audit.Entry{Actor: actorCLI, Action: actionView, Resource: "r1"})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"ts":"","actor":"cli","action":"view","resource":"r1"}`
	if string(got) != want {
		t.Errorf("wire format changed:\ngot:  %s\nwant: %s", got, want)
	}
}

// TestEntry_RoundTrip_OldFormat proves entries written by the previous
// string-based implementation parse into a real time.Time and re-encode
// byte-for-byte identically.
func TestEntry_RoundTrip_OldFormat(t *testing.T) {
	old := `{"ts":"2026-02-13T14:00:00Z","actor":"mcp","action":"ideas.create","resource":"ideas/abc123"}`

	var entry audit.Entry
	if err := json.Unmarshal([]byte(old), &entry); err != nil {
		t.Fatalf("unmarshal old-format entry: %v", err)
	}

	if !entry.Time.Equal(fixedTime()) {
		t.Errorf("expected parsed time %v, got %v", fixedTime(), entry.Time)
	}
	if entry.Actor != actorMCP || entry.Action != actionCreateIdeas || entry.Resource != resourceIdeasABC {
		t.Errorf("unexpected fields: %+v", entry)
	}

	back, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != old {
		t.Errorf("round-trip changed the wire format:\ngot:  %s\nwant: %s", back, old)
	}
}

func TestEntry_UnmarshalJSON_Timestamps(t *testing.T) {
	t.Run("empty ts yields zero time", func(t *testing.T) {
		var entry audit.Entry
		if err := json.Unmarshal([]byte(`{"ts":"","actor":"cli","action":"view","resource":"r1"}`), &entry); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !entry.Time.IsZero() {
			t.Errorf("expected zero time, got %v", entry.Time)
		}
	})

	t.Run("absent ts yields zero time", func(t *testing.T) {
		var entry audit.Entry
		if err := json.Unmarshal([]byte(`{"actor":"cli","action":"view","resource":"r1"}`), &entry); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !entry.Time.IsZero() {
			t.Errorf("expected zero time, got %v", entry.Time)
		}
	})

	t.Run("timezone offsets are preserved", func(t *testing.T) {
		var entry audit.Entry
		if err := json.Unmarshal([]byte(`{"ts":"2026-02-13T15:00:00+01:00","actor":"cli","action":"view","resource":"r1"}`), &entry); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !entry.Time.Equal(fixedTime()) {
			t.Errorf("expected instant %v, got %v", fixedTime(), entry.Time)
		}
	})

	t.Run("malformed ts is an error", func(t *testing.T) {
		var entry audit.Entry
		err := json.Unmarshal([]byte(`{"ts":"not-a-timestamp","actor":"cli","action":"view","resource":"r1"}`), &entry)
		if err == nil {
			t.Fatal("expected error for malformed timestamp, got nil")
		}
	})
}

// TestFileWriter_OverwritesCallerTime documents the Writer contract: the
// writer owns the timestamp and any caller-supplied Time is replaced.
func TestFileWriter_OverwritesCallerTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(fixedTime))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	}()

	err = w.Log(context.Background(), audit.Entry{
		Time:     time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC), // Must be ignored.
		Actor:    actorMCP,
		Action:   actionCreateIdeas,
		Resource: resourceIdeasABC,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(got, []byte(`"ts":"2026-02-13T14:00:00Z"`)) {
		t.Errorf("expected writer clock timestamp in log, got: %s", got)
	}
	if bytes.Contains(got, []byte("1999")) {
		t.Errorf("caller-supplied timestamp leaked into the log: %s", got)
	}
}

// TestReadScorecard_ParsesOldFormatFiles proves log files written by the
// previous string-based implementation still parse.
func TestReadScorecard_ParsesOldFormatFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	// Byte-for-byte lines as produced before the time.Time migration.
	oldLog := `{"ts":"2026-02-13T14:00:00Z","actor":"alice","action":"view","resource":"r1"}
{"ts":"2026-02-13T14:01:00Z","actor":"alice","action":"view","resource":"r1"}
{"ts":"2026-02-13T14:02:00Z","actor":"bob","action":"edit","resource":"r2"}
`
	if err := os.WriteFile(path, []byte(oldLog), 0o600); err != nil {
		t.Fatal(err)
	}

	sc, err := file.ReadScorecard(path)
	if err != nil {
		t.Fatal(err)
	}

	if sc.Total != 3 {
		t.Errorf("expected total 3, got %d", sc.Total)
	}
	if sc.ByActor["alice"] != 2 || sc.ByActor["bob"] != 1 {
		t.Errorf("unexpected ByActor counts: %+v", sc.ByActor)
	}
}

func TestFileWriter_Log(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(fixedTime))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	}()

	err = w.Log(context.Background(), audit.Entry{
		Actor:    actorMCP,
		Action:   actionCreateIdeas,
		Resource: resourceIdeasABC,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"ts":"2026-02-13T14:00:00Z","actor":"mcp","action":"ideas.create","resource":"ideas/abc123"}` + "\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFileWriter_AppendsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(fixedTime))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	}()

	_ = w.Log(context.Background(), audit.Entry{Actor: actorCLI, Action: actionCreateIdeas, Resource: "ideas/a"})
	_ = w.Log(context.Background(), audit.Entry{Actor: actorMCP, Action: "ideas.delete", Resource: "ideas/b"})

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestFileWriter_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(fixedTime))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	}()

	_ = w.Log(context.Background(), audit.Entry{Actor: "api", Action: actionCreateIdeas, Resource: "ideas/x"})

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("audit log not created: %v", err)
	}
}

func TestFileWriter_InvalidPath(t *testing.T) {
	_, err := file.NewFileWriter("/nonexistent/dir/audit.log")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestFileWriter_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Each entry is ~100 bytes. Set max to 250 so 2-3 entries fit before rotation.
	w, err := file.NewFileWriter(path,
		file.WithClock(fixedTime),
		file.WithMaxLogSize(250),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	}()

	// Write enough entries to exceed 250 bytes.
	for range 5 {
		_ = w.Log(context.Background(), audit.Entry{
			Actor:    actorCLI,
			Action:   actionCreateIdeas,
			Resource: "ideas/test-rotation",
		})
	}

	// After rotation, the backup file should exist.
	backupPath := path + ".1"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file not created after rotation: %v", err)
	}

	// The current log should be smaller than the backup.
	currentInfo, _ := os.Stat(path)
	backupInfo, _ := os.Stat(backupPath)

	if currentInfo.Size() >= backupInfo.Size() {
		t.Errorf("current (%d bytes) should be smaller than backup (%d bytes)",
			currentInfo.Size(), backupInfo.Size())
	}
}

func TestReadScorecard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	w, err := file.NewFileWriter(path, file.WithClock(fixedTime))
	if err != nil {
		t.Fatal(err)
	}

	_ = w.Log(context.Background(), audit.Entry{Actor: "alice", Action: actionView, Resource: "r1"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "alice", Action: actionView, Resource: "r1"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "bob", Action: "edit", Resource: "r2"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "bob", Action: actionView, Resource: "r3"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "charlie", Action: "delete", Resource: "r4"})

	_ = w.Close()

	sc, err := file.ReadScorecard(path)
	if err != nil {
		t.Fatal(err)
	}

	if sc.Total != 5 {
		t.Errorf("expected total 5, got %d", sc.Total)
	}

	if sc.ByActor["alice"] != 2 || sc.ByActor["bob"] != 2 || sc.ByActor["charlie"] != 1 {
		t.Errorf("unexpected ByActor counts: %+v", sc.ByActor)
	}

	if sc.ByAction["view"] != 3 || sc.ByAction["edit"] != 1 || sc.ByAction["delete"] != 1 {
		t.Errorf("unexpected ByAction counts: %+v", sc.ByAction)
	}

	// Check top resources.
	if len(sc.TopResources) == 0 || !strings.HasPrefix(sc.TopResources[0], "r1") {
		t.Errorf("expected r1 to be top resource, got: %+v", sc.TopResources)
	}
}
