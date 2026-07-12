package audit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/observability/audit"
	"github.com/duizendstra/alexandria/go/observability/audit/file"
)

var fixedTime = func() time.Time {
	return time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
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
		Actor:    "mcp",
		Action:   "ideas.create",
		Resource: "ideas/abc123",
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

	_ = w.Log(context.Background(), audit.Entry{Actor: "cli", Action: "ideas.create", Resource: "ideas/a"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "mcp", Action: "ideas.delete", Resource: "ideas/b"})

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

	_ = w.Log(context.Background(), audit.Entry{Actor: "api", Action: "ideas.create", Resource: "ideas/x"})

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
			Actor:    "cli",
			Action:   "ideas.create",
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

	_ = w.Log(context.Background(), audit.Entry{Actor: "alice", Action: "view", Resource: "r1"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "alice", Action: "view", Resource: "r1"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "bob", Action: "edit", Resource: "r2"})
	_ = w.Log(context.Background(), audit.Entry{Actor: "bob", Action: "view", Resource: "r3"})
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

	// Check top resources
	if len(sc.TopResources) == 0 || !strings.HasPrefix(sc.TopResources[0], "r1") {
		t.Errorf("expected r1 to be top resource, got: %+v", sc.TopResources)
	}
}
