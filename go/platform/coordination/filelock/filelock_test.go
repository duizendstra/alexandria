package filelock_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
	"github.com/duizendstra/alexandria/go/platform/coordination/filelock"
)

func TestLocker_AcquireAndRelease(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := &filelock.Locker{Dir: dir}

	release, err := locker.Acquire("subject-a")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	lockPath := filepath.Join(dir, filelock.DefaultLockPrefix+"subject-a")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected lock file at %s: %v", lockPath, err)
	}

	// Secondary acquire should fail with coordination.ErrLocked.
	_, err = locker.Acquire("subject-a")
	if !errors.Is(err, coordination.ErrLocked) {
		t.Fatalf("expected coordination.ErrLocked, got: %v", err)
	}

	// Release primary lock.
	release()

	// Lock file should be removed.
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected lock file to be removed after release, got: %v", err)
	}

	// Secondary acquire should now succeed.
	release2, err := locker.Acquire("subject-a")
	if err != nil {
		t.Fatalf("second Acquire failed: %v", err)
	}
	release2()
}

func TestLocker_ConcurrentExclusion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := &filelock.Locker{Dir: dir}

	const workers = 8
	var acquiredCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			release, err := locker.Acquire("singleton-job")
			if err == nil {
				mu.Lock()
				acquiredCount++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				release()
			} else if !errors.Is(err, coordination.ErrLocked) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	if acquiredCount == 0 {
		t.Fatal("expected at least one worker to acquire lock")
	}
}

func TestLocker_InvalidSubject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := &filelock.Locker{Dir: dir}

	invalidSubjects := []string{
		"",
		".",
		"..",
		"sub/unit",
		"sub\\unit",
		"../escape",
	}

	for _, s := range invalidSubjects {
		_, err := locker.Acquire(s)
		if !errors.Is(err, filelock.ErrBadSubject) {
			t.Errorf("subject %q: expected ErrBadSubject, got: %v", s, err)
		}
	}
}

func TestLocker_CustomPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := &filelock.Locker{
		Dir:    dir,
		Prefix: "custom-lock.",
	}

	release, err := locker.Acquire("job-1")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer release()

	lockPath := filepath.Join(dir, "custom-lock.job-1")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected lock file at %s: %v", lockPath, err)
	}
}

func TestLocker_IdempotentRelease(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := &filelock.Locker{Dir: dir}

	release, err := locker.Acquire("job-idempotent")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// Calling release multiple times should be safe and not panic.
	release()
	release()
	release()
}

func TestLocker_OnSignal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sigCh := make(chan os.Signal, 1)

	locker := &filelock.Locker{
		Dir:     dir,
		Signals: []os.Signal{syscall.SIGUSR1},
		OnSignal: func(s os.Signal) {
			sigCh <- s
		},
	}

	release, err := locker.Acquire("signal-test")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer release()

	// Trigger signal handler.
	_ = syscall.Kill(os.Getpid(), syscall.SIGUSR1)

	select {
	case s := <-sigCh:
		if s != syscall.SIGUSR1 {
			t.Fatalf("expected SIGUSR1, got %v", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for signal callback")
	}

	// Lock file should have been cleaned up by signal handler.
	lockPath := filepath.Join(dir, filelock.DefaultLockPrefix+"signal-test")
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected lock file removed after signal, got: %v", err)
	}
}

func TestLocker_InvalidDir(t *testing.T) {
	t.Parallel()

	// Use a file path where a directory is expected to cause MkdirAll failure.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "blocking-file")
	if err := os.WriteFile(filePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	locker := &filelock.Locker{
		Dir: filepath.Join(filePath, "nested"),
	}

	_, err := locker.Acquire("any-job")
	if err == nil {
		t.Fatal("expected error acquiring lock in invalid directory path, got nil")
	}
}
