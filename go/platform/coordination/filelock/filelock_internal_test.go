package filelock

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

// seedRecord writes a holder record at path the way an abandoned or a live
// claim would look on disk, and reads it back through readHolder so the
// caller holds exactly what a reclaimer would have judged.
func seedRecord(t *testing.T, path string, h coordination.Holder) (judged coordination.Holder, fi os.FileInfo) {
	t.Helper()

	record, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(path, append(record, '\n'), 0o600); err != nil {
		t.Fatalf("seed record: %v", err)
	}

	judged, fi, ok := readHolder(path)
	if !ok {
		t.Fatal("the seeded record must read back")
	}

	return judged, fi
}

// noSetAsideSurvives fails the test if any set-aside name is left in dir.
func noSetAsideSurvives(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ReclaimedSuffix) {
			t.Fatalf("a set-aside file %s survived", e.Name())
		}
	}
}

// TestTakeAsideReclaimsTheRecordItJudged pins the through half of the
// reclaim protocol: the record at the path is still the one that was
// judged, so it is taken, the path is freed for the next claim, and the
// set-aside name does not outlive the reclaim.
func TestTakeAsideReclaimsTheRecordItJudged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subject"+LockSuffix)

	judged, fi := seedRecord(t, path, coordination.Holder{
		PID: 1, Host: "gone", Since: time.Now().UTC().Add(-time.Hour), Purpose: "abandoned work",
	})

	if err := takeAside(path, judged, fi); err != nil {
		t.Fatalf("takeAside over the judged record: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the judged record must be gone, stat = %v", err)
	}
	noSetAsideSurvives(t, dir)
}

// TestTakeAsidePutsBackALiveRecordItDidNotJudge pins the identity check at
// the heart of the reclaim protocol, at the seam where the race actually
// happens: the judged record was read, and by the time it is set aside a
// live claim has replaced it. takeAside must put the live record back
// untouched and report nothing to do — an unconditional remove here is how
// a fresh holder loses its record to a reclaim that judged its
// predecessor.
func TestTakeAsidePutsBackALiveRecordItDidNotJudge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subject"+LockSuffix)

	judged, judgedFi := seedRecord(t, path, coordination.Holder{
		PID: 1, Host: "gone", Since: time.Now().UTC().Add(-time.Hour), Purpose: "abandoned work",
	})

	// The judged holder's record leaves and a live claim replaces it,
	// exactly between the reclaimer's read and its set-aside.
	if err := os.Remove(path); err != nil {
		t.Fatalf("clear the judged record: %v", err)
	}
	_, _ = seedRecord(t, path, coordination.Holder{
		PID: 2, Host: "here", Since: time.Now().UTC(), Purpose: "live work",
	})
	live, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the live record: %v", err)
	}

	if err := takeAside(path, judged, judgedFi); err != nil {
		t.Fatalf("takeAside over a replaced record must report nothing to do, got: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the live record must be back in place: %v", err)
	}
	if !bytes.Equal(after, live) {
		t.Fatalf("the live record came back changed: %q became %q", live, after)
	}
	noSetAsideSurvives(t, dir)
}
