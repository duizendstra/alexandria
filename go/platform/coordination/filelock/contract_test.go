package filelock_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
	"github.com/duizendstra/alexandria/go/platform/coordination/filelock"
)

// The cross-process tests below re-execute this very test binary
// (os.Args[0]) as a child process: goroutines inside one process prove
// nothing about a lock whose whole purpose is to coordinate SEPARATE
// processes — one of which may be killed outright without ever running a
// deferred release.
//
// TestMain reads the marker below before the suite starts. A child sees a
// role, plays it and exits; the parent sees nothing and runs the tests.
const (
	childRoleEnv = "COORDINATION_FILELOCK_CHILD_ROLE"

	childDirEnv     = "COORDINATION_FILELOCK_CHILD_DIR"
	childSubjectEnv = "COORDINATION_FILELOCK_CHILD_SUBJECT"
	childReadyEnv   = "COORDINATION_FILELOCK_CHILD_READY"
	childCounterEnv = "COORDINATION_FILELOCK_CHILD_COUNTER"
	childFenceEnv   = "COORDINATION_FILELOCK_CHILD_FENCE"

	roleHold    = "hold"
	roleCompete = "compete"
)

func TestMain(m *testing.M) {
	switch os.Getenv(childRoleEnv) {
	case roleHold:
		childHold()
	case roleCompete:
		childCompete()
	}

	os.Exit(m.Run())
}

// childStore builds the store a child was told to use. Reclaim is off in a
// child: a child never reclaims, it only holds.
func childStore() (*filelock.Store, coordination.Subject) {
	return &filelock.Store{
			Dir:     os.Getenv(childDirEnv),
			Purpose: "child holding the shared index",
			Options: filelock.Options{PollMin: 5 * time.Millisecond, PollMax: 20 * time.Millisecond},
		},
		coordination.Subject(os.Getenv(childSubjectEnv))
}

// childHold enters the window, announces that it is inside, and then never
// leaves: the parent kills it. This is the holder that crashes.
func childHold() {
	store, subject := childStore()

	_, _, err := store.Acquire(context.Background(), subject)
	if err != nil {
		fmt.Fprintln(os.Stderr, "child hold:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(os.Getenv(childReadyEnv), []byte("inside\n"), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "child ready:", err)
		os.Exit(1)
	}

	// Deliberate: the parent ends this process with SIGKILL, so nothing here
	// ever releases. A bare `select {}` is the wrong way to say that: the Go
	// runtime's deadlock detector sees every goroutine parked with no pending
	// timer, no channel send, nothing anywhere that could ever wake one of
	// them, and kills the process itself with "fatal error: all goroutines
	// are asleep - deadlock!" — the child would die on its own before the
	// parent's SIGKILL ever arrives, which defeats the point of this fixture.
	// A goroutine that keeps a ticker running stays live in the runtime's
	// eyes, so blocking on a channel this hold deliberately never writes to —
	// while that goroutine keeps ticking — blocks the same way forever
	// without ever looking like a deadlock.
	block := make(chan struct{})

	go func() {
		for range time.Tick(time.Second) {
			// Nothing to do here: this goroutine exists only to keep the
			// runtime convinced the process can still make progress, so the
			// deadlock detector leaves the blocked main goroutine alone.
		}
	}()

	<-block
}

// childCompete enters the window, does a read-pause-write on a counter file
// shared by every child, appends the fence it was given, and leaves. The
// pause is what makes an overlap visible: two children inside at once would
// read the same value and one increment would be lost.
func childCompete() {
	if err := compete(); err != nil {
		fmt.Fprintln(os.Stderr, "child compete:", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// compete is childCompete's body, split out so its deferred cleanup actually
// runs: os.Exit skips defers, so the exit has to live above them.
func compete() error {
	store, subject := childStore()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	release, fence, err := store.Acquire(ctx, subject)
	if err != nil {
		return err
	}
	defer release()

	counter := os.Getenv(childCounterEnv)

	n := 0
	if b, err := os.ReadFile(counter); err == nil {
		n, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	}

	time.Sleep(20 * time.Millisecond)

	if err := os.WriteFile(counter, []byte(strconv.Itoa(n+1)+"\n"), 0o600); err != nil {
		return err
	}

	f, err := os.OpenFile(os.Getenv(childFenceEnv), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(f, "%d\n", fence)

	return f.Close()
}

// childCmd builds a re-execution of this test binary in the given role.
func childCmd(t *testing.T, role, dir string, subject coordination.Subject, extra ...string) *exec.Cmd {
	t.Helper()

	// os.Args[0] is this very test binary — the whole point of the pattern.
	cmd := exec.CommandContext(t.Context(), os.Args[0])
	cmd.Env = append(os.Environ(),
		childRoleEnv+"="+role,
		childDirEnv+"="+dir,
		childSubjectEnv+"="+string(subject),
	)
	cmd.Env = append(cmd.Env, extra...)
	cmd.Stderr = os.Stderr

	return cmd
}

// waitForFile blocks until path exists, or fails the test.
func waitForFile(t *testing.T, path string, within time.Duration) {
	t.Helper()

	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("%s never appeared within %v", path, within)
}

// TestCoordination_MutualExclusionAcrossProcesses is the promise this
// package exists for, proven where it actually has to hold: several
// SEPARATE processes queue for one subject, and a counter each of them
// read-pauses-writes while inside ends at exactly the number of processes.
// One overlap loses one increment, and the count says so.
func TestCoordination_MutualExclusionAcrossProcesses(t *testing.T) {
	const children = 4

	dir := t.TempDir()
	store := filepath.Join(dir, "coordination")
	counter := filepath.Join(dir, "counter")
	fences := filepath.Join(dir, "fences")

	var wg sync.WaitGroup
	errs := make(chan error, children)

	for range children {
		wg.Go(func() {
			cmd := childCmd(t, roleCompete, store, testSubject,
				childCounterEnv+"="+counter,
				childFenceEnv+"="+fences,
			)
			if err := cmd.Run(); err != nil {
				errs <- err
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("child process: %v", err)
	}

	b, err := os.ReadFile(counter)
	if err != nil {
		t.Fatalf("counter: %v", err)
	}
	got, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatalf("counter %q: %v", b, err)
	}
	if got != children {
		t.Fatalf("counter = %d after %d processes, want %d — an increment was lost, so two processes were inside the window at once", got, children, children)
	}

	seen := map[string]bool{}
	for line := range strings.FieldsSeq(readFile(t, fences)) {
		if seen[line] {
			t.Fatalf("fence %s was handed out twice: %s", line, readFile(t, fences))
		}
		seen[line] = true
	}
	if len(seen) != children {
		t.Fatalf("got %d distinct fences from %d processes: %s", len(seen), children, readFile(t, fences))
	}
}

// TestCoordination_KilledHolderIsReclaimedOnlyWhenEnabled pins the
// safety-versus-liveness trade, both halves, against a real holder that was
// killed outright and never ran a release: with reclaim off (the zero
// value) a later caller waits until its own context ends, and with reclaim
// on it enters once the record is older than the configured age. The fence
// keeps rising across the reclaim, so the abandoned occupancy and the one
// that took over are still distinguishable afterwards.
func TestCoordination_KilledHolderIsReclaimedOnlyWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "coordination")
	ready := filepath.Join(dir, "ready")

	cmd := childCmd(t, roleHold, storeDir, testSubject, childReadyEnv+"="+ready)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start holder: %v", err)
	}

	waitForFile(t, ready, 10*time.Second)

	// The fixture proves nothing about a killed holder if the holder was
	// already dead on its own — signal 0 sends nothing but still fails if the
	// process is gone, so this pins that childHold's real block, not a
	// deadlock-detector exit, is what the parent is about to kill.
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("holder process not alive just before kill: %v", err)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill holder: %v", err)
	}
	_ = cmd.Wait()

	// The record is still there: nothing released it.
	safe := &filelock.Store{Dir: storeDir, Purpose: "later caller", Options: filelock.Options{PollMin: 5 * time.Millisecond, PollMax: 20 * time.Millisecond}}
	if _, held, err := safe.Holder(testSubject); err != nil || !held {
		t.Fatalf("holder record after the kill: held=%v err=%v — the fixture proves nothing without one", held, err)
	}

	t.Run("reclaim off: the later caller waits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		if _, _, err := safe.Acquire(ctx, testSubject); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("acquire with reclaim off = %v, want a wait that ends on the context", err)
		}
	})

	t.Run("reclaim on: the later caller enters", func(t *testing.T) {
		// The record was written moments ago, so give it an age to exceed.
		time.Sleep(60 * time.Millisecond)

		live := &filelock.Store{
			Dir:     storeDir,
			Purpose: "later caller",
			Options: filelock.Options{PollMin: 5 * time.Millisecond, PollMax: 20 * time.Millisecond, StaleAfter: 50 * time.Millisecond},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		release, fence, err := live.Acquire(ctx, testSubject)
		if err != nil {
			t.Fatalf("acquire with reclaim on: %v", err)
		}
		defer release()

		if fence != 2 {
			t.Fatalf("fence after reclaiming the first occupancy = %d, want 2 — the counter must keep rising across a reclaim", fence)
		}
	})
}

// TestCoordination_FenceIsStrictlyIncreasing pins the fence's whole promise:
// per subject, never repeated, always higher than the occupancy before it,
// and independent of any other subject's counter.
func TestCoordination_FenceIsStrictlyIncreasing(t *testing.T) {
	store := newStore(t, filelock.Options{})

	var last uint64
	for i := range 5 {
		release, fence, err := store.Acquire(context.Background(), testSubject)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		release()

		if fence <= last {
			t.Fatalf("fence %d after %d — a fence must be strictly higher than the occupancy before it", fence, last)
		}
		last = fence
	}

	if last != 5 {
		t.Fatalf("fence after five occupancies = %d, want 5", last)
	}

	// A second subject counts on its own.
	release, other, err := store.Acquire(context.Background(), "other-index")
	if err != nil {
		t.Fatalf("acquire other subject: %v", err)
	}
	release()

	if other != 1 {
		t.Fatalf("first fence of a fresh subject = %d, want 1 — counters are per subject", other)
	}

	// The counter outlives the occupancy: releasing must not remove it.
	fencePath, err := store.FencePath(testSubject)
	if err != nil {
		t.Fatalf("fence path: %v", err)
	}
	if _, err := os.Stat(fencePath); err != nil {
		t.Fatalf("the counter file must survive release: %v", err)
	}
}

// TestCoordination_HolderRecordIsReadableJSON pins what an operator finds in
// the store while a window is held: one JSON object, the four documented
// fields, this process, and the purpose the store was given — readable with
// nothing but `cat`.
func TestCoordination_HolderRecordIsReadableJSON(t *testing.T) {
	store := newStore(t, filelock.Options{})

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	raw := readFile(t, lockPath(t, store))

	var holder coordination.Holder
	if err := json.Unmarshal([]byte(raw), &holder); err != nil {
		t.Fatalf("the holder record must be readable JSON, got %q: %v", raw, err)
	}

	if holder.PID != os.Getpid() {
		t.Errorf("record pid = %d, want this process %d", holder.PID, os.Getpid())
	}
	if holder.Host == "" {
		t.Error("record host must not be empty")
	}
	if holder.Purpose != testPurpose {
		t.Errorf("record purpose = %q, want the store's stated purpose", holder.Purpose)
	}
	if time.Since(holder.Since) > time.Minute || holder.Since.IsZero() {
		t.Errorf("record since = %v, want the moment the window was entered", holder.Since)
	}

	fromStore, held, err := store.Holder(testSubject)
	if err != nil || !held {
		t.Fatalf("Store.Holder = held %v, err %v — it must report the same record", held, err)
	}
	if fromStore.PID != holder.PID || !fromStore.Since.Equal(holder.Since) {
		t.Fatalf("Store.Holder = %+v, want the record on disk %+v", fromStore, holder)
	}

	release()

	if _, held, err := store.Holder(testSubject); err != nil || held {
		t.Fatalf("after release Store.Holder = held %v, err %v, want held false", held, err)
	}
}

// readFile reads path or fails the test.
func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}
