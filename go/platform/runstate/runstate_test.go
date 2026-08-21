package runstate_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/runstate"
)

const (
	subject     = "job-alpha"
	other       = "job-beta"
	fingerprint = "1a2b3c4d"
)

func TestLeaseValid(t *testing.T) {
	now := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)
	window := time.Hour

	tests := []struct {
		name  string
		lease runstate.Lease
		want  bool
	}{
		{"fresh", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now.Add(-time.Minute)}, true},
		{"issued now", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now}, true},
		{"just inside the window", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now.Add(-59 * time.Minute)}, true},
		{"exactly at the window", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now.Add(-window)}, false},
		{"expired", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now.Add(-2 * time.Hour)}, false},
		{"other subject", runstate.Lease{Subject: other, Fingerprint: fingerprint, IssuedAt: now}, false},
		{"other fingerprint", runstate.Lease{Subject: subject, Fingerprint: "deadbeef", IssuedAt: now}, false},
		{"no fingerprint", runstate.Lease{Subject: subject, IssuedAt: now}, false},
		{"dated in the future", runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: now.Add(time.Minute)}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.lease.Valid(now, subject, fingerprint, window); got != tc.want {
				t.Fatalf("Valid = %v, want %v", got, tc.want)
			}
		})
	}
}

// An empty fingerprint on both sides must not accidentally match.
func TestLeaseValidRejectsEmptyFingerprintOnBothSides(t *testing.T) {
	now := time.Now()
	lease := runstate.Lease{Subject: subject, IssuedAt: now}

	if lease.Valid(now, subject, "", time.Hour) {
		t.Fatal("a lease without a fingerprint may never be valid")
	}
}

func TestLeaseStoreRoundTrip(t *testing.T) {
	store := &runstate.LeaseStore{Dir: filepath.Join(t.TempDir(), "state")}

	if _, ok, err := store.Load(subject); err != nil || ok {
		t.Fatalf("a missing lease is not an error: ok=%v err=%v", ok, err)
	}

	issued := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)
	if err := store.Save(runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: issued}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, ok, err := store.Load(subject)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}

	if got.Subject != subject || got.Fingerprint != fingerprint || !got.IssuedAt.Equal(issued) {
		t.Fatalf("lease = %+v", got)
	}

	path, err := store.Path(subject)
	if err != nil {
		t.Fatalf("path: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %v, want 0600", fi.Mode().Perm())
	}

	if err := store.Consume(subject); err != nil {
		t.Fatalf("consume: %v", err)
	}

	if _, ok, _ := store.Load(subject); ok {
		t.Fatal("a consumed lease is gone")
	}

	if err := store.Consume(subject); err != nil {
		t.Fatalf("consume must be idempotent: %v", err)
	}
}

func TestLeaseStoreSaveIsAtomic(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	store := &runstate.LeaseStore{Dir: dir}

	for range 3 {
		if err := store.Save(runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}

	names, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 1 {
		t.Fatalf("after three saves the directory holds %v; the write must leave no temporary behind", names)
	}
}

func TestLeaseStoreIgnoresAnUnreadableLease(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	store := &runstate.LeaseStore{Dir: dir}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	path, err := store.Path(subject)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	lease, ok, err := store.Load(subject)
	if err != nil {
		t.Fatalf("a broken lease must not be an error: %v", err)
	}

	if ok || lease.Fingerprint != "" {
		t.Fatal("a broken lease counts as no lease")
	}
}

func TestLeaseStorePrefix(t *testing.T) {
	dir := t.TempDir()
	store := &runstate.LeaseStore{Dir: dir, Prefix: "gate-token."}

	path, err := store.Path(subject)
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(path) != "gate-token."+subject+".json" {
		t.Fatalf("path = %s", path)
	}
}

func TestLeaseIsReadableJSON(t *testing.T) {
	store := &runstate.LeaseStore{Dir: t.TempDir()}
	issued := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)

	if err := store.Save(runstate.Lease{Subject: subject, Fingerprint: fingerprint, IssuedAt: issued}); err != nil {
		t.Fatal(err)
	}

	path, _ := store.Path(subject)

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("the lease must be plain JSON: %v", err)
	}

	for _, field := range []string{"subject", "fingerprint", "issued_at"} {
		if _, ok := raw[field]; !ok {
			t.Fatalf("the lease is missing %q: %s", field, b)
		}
	}
}

func TestSubjectsThatWouldEscapeAreRefused(t *testing.T) {
	dir := t.TempDir()
	store := &runstate.LeaseStore{Dir: dir}
	locker := &runstate.Locker{Dir: dir, OnSignal: func(os.Signal) {}}

	for _, bad := range []string{"", ".", "..", "../escape", "sub/dir", "a/../b"} {
		t.Run("subject "+bad, func(t *testing.T) {
			if _, err := store.Path(bad); !errors.Is(err, runstate.ErrBadSubject) {
				t.Fatalf("Path(%q) must be refused, got %v", bad, err)
			}

			if _, _, err := store.Load(bad); !errors.Is(err, runstate.ErrBadSubject) {
				t.Fatalf("Load(%q) must be refused, got %v", bad, err)
			}

			if err := store.Save(runstate.Lease{Subject: bad}); !errors.Is(err, runstate.ErrBadSubject) {
				t.Fatalf("Save(%q) must be refused, got %v", bad, err)
			}

			if err := store.Consume(bad); !errors.Is(err, runstate.ErrBadSubject) {
				t.Fatalf("Consume(%q) must be refused, got %v", bad, err)
			}

			if _, err := locker.Acquire(bad); !errors.Is(err, runstate.ErrBadSubject) {
				t.Fatalf("Acquire(%q) must be refused, got %v", bad, err)
			}
		})
	}
}

// A subject that is an e-mail address, a dotted name or a dashed identifier is
// ordinary and must be accepted.
func TestOrdinarySubjectsAreAccepted(t *testing.T) {
	store := &runstate.LeaseStore{Dir: t.TempDir()}

	for _, good := range []string{"job-alpha", "user@example.test", "team.reporting", "run_2030"} {
		if _, err := store.Path(good); err != nil {
			t.Fatalf("Path(%q) = %v", good, err)
		}
	}
}

func TestLockerExcludesASecondRun(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	locker := &runstate.Locker{Dir: dir, OnSignal: func(os.Signal) {}}

	release, err := locker.Acquire(subject)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "lock."+subject)); err != nil {
		t.Fatalf("the lock file is missing: %v", err)
	}

	if _, err := locker.Acquire(subject); !errors.Is(err, runstate.ErrLocked) {
		t.Fatalf("a second run must be refused with ErrLocked, got %v", err)
	}

	// Another subject is free.
	releaseOther, err := locker.Acquire(other)
	if err != nil {
		t.Fatalf("the lock is per subject: %v", err)
	}

	releaseOther()

	release()

	if _, err := os.Stat(filepath.Join(dir, "lock."+subject)); !os.IsNotExist(err) {
		t.Fatal("the lock must be cleaned up")
	}

	release() // release is safe to call twice.

	// After release the subject can be locked again.
	again, err := locker.Acquire(subject)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}

	again()
}

func TestLockerPrefix(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	locker := &runstate.Locker{Dir: dir, Prefix: "run-", OnSignal: func(os.Signal) {}}

	release, err := locker.Acquire(subject)
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	if _, err := os.Stat(filepath.Join(dir, "run-"+subject)); err != nil {
		t.Fatalf("the prefix must be used: %v", err)
	}
}

func TestLockerReportsAnUnusableDirectory(t *testing.T) {
	// A file where the directory should be: MkdirAll cannot succeed.
	base := t.TempDir()
	blocked := filepath.Join(base, "state")

	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	locker := &runstate.Locker{Dir: blocked, OnSignal: func(os.Signal) {}}
	if _, err := locker.Acquire(subject); err == nil {
		t.Fatal("an unusable lock directory must produce an error")
	}
}

func TestLeaseStoreReportsAnUnusableDirectory(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "state")

	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &runstate.LeaseStore{Dir: blocked}
	if err := store.Save(runstate.Lease{Subject: subject, Fingerprint: fingerprint}); err == nil {
		t.Fatal("an unusable lease directory must produce an error")
	}

	if _, _, err := store.Load(subject); err == nil {
		t.Fatal("an unreadable lease path must produce an error")
	}
}

func TestLockerSignalCleansUpTheLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")

	seen := make(chan os.Signal, 1)
	locker := &runstate.Locker{
		Dir:     dir,
		Signals: []os.Signal{syscallSIGUSR1},
		OnSignal: func(s os.Signal) {
			seen <- s
		},
	}

	release, err := locker.Acquire(subject)
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	raise(t, syscallSIGUSR1)

	select {
	case <-seen:
	case <-time.After(5 * time.Second):
		t.Fatal("the signal handler did not run")
	}

	if _, err := os.Stat(filepath.Join(dir, "lock."+subject)); !os.IsNotExist(err) {
		t.Fatal("a signal must clean up the lock")
	}
}
