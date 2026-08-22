package filelock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
	"github.com/duizendstra/alexandria/go/platform/coordination/filelock"
)

// testSubject is the subject every test below queues for, and testPurpose is
// what each store states it is holding the window for.
const (
	testSubject coordination.Subject = "shared-index"
	testPurpose string               = "updating the shared index"
)

// newStore is a store in a fresh directory, with reclaim off (the default)
// unless a test states otherwise, and poll bounds short enough that a
// blocked waiter is picked up promptly inside a test's patience.
func newStore(t *testing.T, opts filelock.Options) *filelock.Store {
	t.Helper()

	if opts.PollMin == 0 && opts.PollMax == 0 {
		opts.PollMin, opts.PollMax = 5*time.Millisecond, 20*time.Millisecond
	}

	return &filelock.Store{Dir: t.TempDir(), Purpose: testPurpose, Options: opts}
}

// lockPath is the store's holder-record path for testSubject.
func lockPath(t *testing.T, s *filelock.Store) string {
	t.Helper()

	path, err := s.LockPath(testSubject)
	if err != nil {
		t.Fatalf("lock path: %v", err)
	}

	return path
}

// TestAcquireExcludesASecondCaller pins the basic mutual exclusion: a second
// Acquire for the same subject blocks until the first release runs, then
// succeeds, and the holder record is gone once both are released.
func TestAcquireExcludesASecondCaller(t *testing.T) {
	store := newStore(t, filelock.Options{})
	path := lockPath(t, store)

	release1, fence1, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if fence1 == 0 {
		t.Fatal("a successful acquire must carry a non-zero fence")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("holder record missing: %v", err)
	}

	second := make(chan struct{})
	go func() {
		release2, _, err := store.Acquire(context.Background(), testSubject)
		if err != nil {
			t.Errorf("second acquire: %v", err)

			return
		}
		release2()
		close(second)
	}()

	// The second caller must not have finished yet: it is polling, blocked
	// behind the first holder.
	select {
	case <-second:
		t.Fatal("a second Acquire for the same subject must block while the first is held")
	case <-time.After(150 * time.Millisecond):
	}

	release1()

	select {
	case <-second:
	case <-time.After(2 * time.Second):
		t.Fatal("the second Acquire never completed after release")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the holder record must be gone once its holder releases")
	}
}

// TestAcquireIsIdempotentOnRelease pins that calling release more than once,
// and deferring it unconditionally, is safe.
func TestAcquireIsIdempotentOnRelease(t *testing.T) {
	store := newStore(t, filelock.Options{})
	path := lockPath(t, store)

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	release()
	release()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the holder record must be removed after release")
	}
}

// TestAcquireRespectsContextCancellation pins that a waiter gives up (with an
// error wrapping ctx.Err()) once its context ends, rather than blocking
// forever behind a window nobody is going to leave.
func TestAcquireRespectsContextCancellation(t *testing.T) {
	store := newStore(t, filelock.Options{})

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, fence, err := store.Acquire(ctx, testSubject)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("acquire error = %v, want one wrapping context.DeadlineExceeded", err)
	}
	if fence != 0 {
		t.Fatalf("a failed acquire must carry fence 0, got %d", fence)
	}
}

// TestAcquireIsMutuallyExclusiveUnderConcurrency runs many goroutines at the
// same subject and pins that a shared counter, incremented and decremented
// only while a goroutine holds the window, never shows more than one holder
// at once — the actual guarantee this package exists for.
func TestAcquireIsMutuallyExclusiveUnderConcurrency(t *testing.T) {
	store := newStore(t, filelock.Options{})

	var (
		mu       sync.Mutex
		inFlight int
		maxWidth int
		wg       sync.WaitGroup
	)

	const goroutines = 8

	for range goroutines {
		wg.Go(func() {
			release, _, err := store.Acquire(context.Background(), testSubject)
			if err != nil {
				t.Errorf("acquire: %v", err)

				return
			}
			defer release()

			mu.Lock()
			inFlight++
			if inFlight > maxWidth {
				maxWidth = inFlight
			}
			mu.Unlock()

			time.Sleep(5 * time.Millisecond)

			mu.Lock()
			inFlight--
			mu.Unlock()
		})
	}

	wg.Wait()

	if maxWidth != 1 {
		t.Fatalf("max concurrent holders = %d, want 1", maxWidth)
	}
}

// TestAcquireReclaimsAnAbandonedLock pins the opt-in staleness safeguard: a
// holder record left behind by a holder that never released does not block
// every later caller forever ONCE reclaim is switched on. The same record
// with reclaim off (the default) is waited on instead — that half is the
// point of the option and is pinned in
// TestCoordination_KilledHolderIsReclaimedOnlyWhenEnabled with a real
// killed process.
func TestAcquireReclaimsAnAbandonedLock(t *testing.T) {
	store := newStore(t, filelock.Options{StaleAfter: time.Minute})
	path := lockPath(t, store)

	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatalf("store dir: %v", err)
	}

	abandoned := coordination.Holder{
		PID:     99999,
		Host:    "gone",
		Since:   time.Now().UTC().Add(-time.Hour),
		Purpose: testPurpose,
	}
	record, err := json.Marshal(abandoned)
	if err != nil {
		t.Fatalf("marshal abandoned record: %v", err)
	}
	if err := os.WriteFile(path, record, 0o600); err != nil {
		t.Fatalf("seed abandoned record: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	release, fence, err := store.Acquire(ctx, testSubject)
	if err != nil {
		t.Fatalf("acquire over an abandoned record: %v", err)
	}
	if fence == 0 {
		t.Fatal("a reclaimed window still owes its holder a fence")
	}
	release()

	if stray := strayFiles(t, store.Dir); len(stray) > 0 {
		t.Fatalf("the reclaim left %v behind — a set-aside name must not survive a successful reclaim", stray)
	}
}

// strayFiles lists everything in a store directory that is neither a holder
// record nor a counter — the only two files a store is documented to own. A
// file an acquisition staged its record in and failed to clean up shows up
// here, and one left behind per attempt is what a long wait would turn into
// a slow leak.
func strayFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read store dir %s: %v", dir, err)
	}

	var stray []string

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), filelock.LockSuffix) || strings.HasSuffix(e.Name(), filelock.FenceSuffix) {
			continue
		}

		stray = append(stray, e.Name())
	}

	return stray
}

// TestHolderRecordIsNeverPartiallyVisible pins what the holder record's
// existence is worth: the moment the path exists at all it carries a
// complete, parseable record naming this process. The record is staged
// elsewhere and published under its real name in one step, so there is no
// moment at which the name exists and the content does not — and the staging
// file does not outlive the acquisition that used it.
func TestHolderRecordIsNeverPartiallyVisible(t *testing.T) {
	store := newStore(t, filelock.Options{})

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	raw, err := os.ReadFile(lockPath(t, store))
	if err != nil {
		t.Fatalf("the holder record must exist while the window is held: %v", err)
	}

	var holder coordination.Holder
	if err := json.Unmarshal(raw, &holder); err != nil {
		t.Fatalf("the holder record must be complete the moment it exists, got %q: %v", raw, err)
	}

	if holder.PID != os.Getpid() || holder.Host == "" || holder.Since.IsZero() || holder.Purpose != testPurpose {
		t.Fatalf("holder record = %+v, want this process, a host, the instant it was entered and the store's stated purpose", holder)
	}

	if stray := strayFiles(t, store.Dir); len(stray) > 0 {
		t.Fatalf("the store kept %v — the file the record was staged in must not survive the acquisition", stray)
	}
}

// TestConcurrentAcquireStillAdmitsExactlyOne pins that publishing the
// record by a primitive that REFUSES an existing target keeps the exclusion
// the whole package is made of: many goroutines at one subject, never more
// than one inside. A publish that replaced its target instead would let a
// second caller overwrite a live claim, and the width would exceed one.
//
// A reader watches the record throughout: every read either finds no file at
// all or finds a complete record, never a name whose content is still on its
// way.
func TestConcurrentAcquireStillAdmitsExactlyOne(t *testing.T) {
	store := newStore(t, filelock.Options{})
	path := lockPath(t, store)

	var (
		mu       sync.Mutex
		inFlight int
		maxWidth int
		wg       sync.WaitGroup
	)

	const goroutines = 8

	done := make(chan struct{})
	watched := make(chan struct{})

	go func() {
		defer close(watched)

		for {
			select {
			case <-done:
				return
			default:
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				continue // No claim right now: nothing to check.
			}

			var h coordination.Holder
			if err := json.Unmarshal(raw, &h); err != nil {
				t.Errorf("a holder record was visible half-written: %q does not parse: %v", raw, err)

				return
			}
		}
	}()

	for range goroutines {
		wg.Go(func() {
			release, _, err := store.Acquire(context.Background(), testSubject)
			if err != nil {
				t.Errorf("acquire: %v", err)

				return
			}
			defer release()

			mu.Lock()
			inFlight++
			if inFlight > maxWidth {
				maxWidth = inFlight
			}
			mu.Unlock()

			time.Sleep(5 * time.Millisecond)

			mu.Lock()
			inFlight--
			mu.Unlock()
		})
	}

	wg.Wait()
	close(done)
	<-watched

	if maxWidth != 1 {
		t.Fatalf("max concurrent holders = %d, want 1 — publishing the record must refuse an existing claim, not replace it", maxWidth)
	}
}

// TestExternallyTruncatedRecordIsStillNeverReclaimed pins that the
// fail-safe survives the change: a record that cannot be read is still a
// claim, and age is not allowed to argue otherwise. Nothing in this package
// can produce such a record any more, so one that appears came from outside
// it — damage, or a hand that does not go through this package — which is
// exactly the case where reclaiming what cannot be identified is the more
// dangerous reading.
func TestExternallyTruncatedRecordIsStillNeverReclaimed(t *testing.T) {
	store := newStore(t, filelock.Options{PollMin: 5 * time.Millisecond, PollMax: 20 * time.Millisecond, StaleAfter: 20 * time.Millisecond})
	path := lockPath(t, store)

	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatalf("store dir: %v", err)
	}

	// Written from outside the package, empty, and left to age well past the
	// configured reclaim age.
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed an unreadable record: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	if _, _, err := store.Acquire(ctx, testSubject); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("acquire over an unreadable record = %v, want a wait that ends on the context — an unreadable claim is never reclaimed on age", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the unreadable record must still be there afterwards: %v", err)
	}
	if len(raw) != 0 {
		t.Fatalf("the unreadable record was rewritten by a caller that never entered: %q", raw)
	}
}

// TestContendedAcquireLeavesNoTempFiles pins that a long wait costs the
// directory nothing: every attempt that is refused because somebody else
// holds the window stages a record and must clean it up again, so dozens of
// refused attempts leave the store holding exactly the two files it owns.
func TestContendedAcquireLeavesNoTempFiles(t *testing.T) {
	store := newStore(t, filelock.Options{PollMin: 5 * time.Millisecond, PollMax: 10 * time.Millisecond})

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if _, _, err := store.Acquire(ctx, testSubject); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the contended acquire = %v, want a wait that polls and then ends on the context", err)
	}

	if stray := strayFiles(t, store.Dir); len(stray) > 0 {
		t.Fatalf("a wait of many refused attempts left %v behind — each attempt must remove the file it staged", stray)
	}
}

// TestContendedAcquireIsNotBornStale pins WHEN the record is stamped,
// which is a separate property from how it is published and is not implied by
// any of the tests above. The record carries the instant the window was
// entered, and reclaim reads that instant to decide whether a holder has been
// abandoned. Staging the record once and reusing it across a long wait would
// look like an obvious saving — one create and one fsync for the whole wait
// instead of one per poll — and would stamp the record at the FIRST attempt
// instead of the winning one. A caller that queued for longer than the
// reclaim age would then enter a window whose record is already older than
// that age: the next caller judges a live holder abandoned, takes the window
// from underneath it, and nothing anywhere reports that two callers are
// inside. So the record is staged per attempt, and that cost is the price of
// the stamp being true.
//
// The waiter deliberately has reclaim OFF and the caller behind it has it on.
// A waiter that could reclaim would take the window from the holder it is
// queued behind as soon as the wait passed the reclaim age, and would then
// never have waited long enough for the stamp to matter.
func TestContendedAcquireIsNotBornStale(t *testing.T) {
	// The wait is a large multiple of the reclaim age, so a record stamped
	// at the first attempt is unmistakably reclaimable by the time it is
	// linked; the last caller's patience is a small fraction of it, so the
	// only way it can enter is by judging a just-written record abandoned.
	const (
		reclaimAge = 500 * time.Millisecond
		contention = 5 * reclaimAge
		patience   = reclaimAge / 5

		// An attempt that was already staging its record when the window
		// opened stamped it slightly before that instant — the record is
		// written first and published second, so its stamp legitimately
		// precedes the opening by however long the create and the fsync
		// took (~6ms on a laptop, 77ms observed on ubuntu CI under -race) —
		// and links slightly after. The failure being pinned puts the stamp
		// a whole contention early, so half a contention still separates a
		// slow staging attempt from a record stamped at the first attempt.
		inFlight = contention / 2

		// The last caller's reclaim age must cover the whole window from the
		// winner's stamp to that caller's LAST look at the record, and that
		// window is much wider than the staging gap above. The stamp is
		// taken before the record's fsync and the directory fsync; the
		// winner then advances the fence (a fsynced write and another
		// directory fsync) before its Acquire returns; the test reads the
		// record back; and the last caller spends its whole patience
		// polling, then makes one more non-interruptible attempt (another
		// fsync) before it consults its context. Five fsyncs and the full
		// patience, under -race, on a runner that may stall. Sizing this to
		// the reclaim age — as before — covered one fsync plus some latency
		// and spent a fifth of the budget on patience by design, so a slow
		// disk pushed the record past it: the last caller then rightly
		// reclaimed a record older than its StaleAfter and the test failed
		// on a premise it never checked. The in-flight margin is the budget
		// this test already grants a slow disk, and it is still half a
		// contention below the failure being pinned, so a record stamped at
		// the first attempt remains unmistakably reclaimable.
		lastCallerStaleAfter = inFlight
	)

	waiter := newStore(t, filelock.Options{PollMin: time.Millisecond, PollMax: 2 * time.Millisecond})
	path := lockPath(t, waiter)

	releaseFirst, _, err := waiter.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	type outcome struct {
		release func()
		err     error
	}
	won := make(chan outcome, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		release, _, err := waiter.Acquire(ctx, testSubject)
		won <- outcome{release: release, err: err}
	}()

	time.Sleep(contention)

	opened := time.Now().UTC()
	releaseFirst()

	var second outcome
	select {
	case second = <-won:
	case <-time.After(10 * time.Second):
		t.Fatal("the contended acquire never finished")
	}
	if second.err != nil {
		t.Fatalf("contended acquire: %v", second.err)
	}
	defer second.release()

	holder, ok, err := waiter.Holder(testSubject)
	if err != nil {
		t.Fatalf("read the winner's record: %v", err)
	}
	if !ok {
		t.Fatal("the winner holds the window but its record is not there")
	}
	if !holder.Since.After(opened.Add(-inFlight)) {
		t.Fatalf("the winner's record is stamped %v, %v before the window opened "+
			"at %v — it must be stamped on the attempt that wins, not on the "+
			"first attempt of a long wait",
			holder.Since, opened.Sub(holder.Since), opened)
	}

	// And the consequence, from outside: the next caller must find a live
	// claim, not something it is entitled to take away.
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the winner's record from disk: %v", err)
	}

	reclaimer := &filelock.Store{
		Dir:     waiter.Dir,
		Purpose: testPurpose,
		Options: filelock.Options{PollMin: time.Millisecond, PollMax: 2 * time.Millisecond, StaleAfter: lastCallerStaleAfter},
	}

	ctx, cancel := context.WithTimeout(context.Background(), patience)
	defer cancel()

	releaseLast, _, lastErr := reclaimer.Acquire(ctx, testSubject)
	if lastErr == nil {
		defer releaseLast()
	}

	// The assertion below rests on the record being younger than the last
	// caller's reclaim age at its last look. Measure that instead of
	// assuming it: the look happened before Acquire returned, so the gap to
	// now bounds it from above, and a gap beyond the budget means the host
	// stalled for longer than the test allows — nothing about the stamp —
	// so the test has no verdict to give.
	if gap := time.Now().UTC().Sub(holder.Since); gap > lastCallerStaleAfter {
		t.Skipf("the winner's record was stamped %v before the last caller's last look, "+
			"beyond the %v it may reclaim at: the host stalled for longer than the "+
			"budget, so the last caller was entitled to what it did (it returned %v)",
			gap.Round(time.Millisecond), lastCallerStaleAfter, lastErr)
	}

	if !errors.Is(lastErr, context.DeadlineExceeded) {
		t.Fatalf("the next caller = %v, want a wait: a record written moments ago is not an abandoned one", lastErr)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the winner's record after the next caller's wait: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("the winner's record changed under it: %q became %q", before, after)
	}
}

// TestReleaseRemovesOnlyItsOwnRecord pins what a release is allowed to
// remove: its OWN record, identified by file, never whatever happens to be
// at the path. The scenario is a reclaim overlap — a holder still inside
// while its record is judged abandoned, removed, and replaced by a
// successor's. The overlapped holder's release must leave the successor's
// record exactly where it is: removing it would hand the window to a third
// caller while the successor still believes it is inside.
func TestReleaseRemovesOnlyItsOwnRecord(t *testing.T) {
	store := newStore(t, filelock.Options{})
	path := lockPath(t, store)

	release1, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Simulate the reclaim that judged the first holder abandoned: its
	// record disappears while it is still inside...
	if err := os.Remove(path); err != nil {
		t.Fatalf("simulate the reclaim: %v", err)
	}

	// ...and a successor claims the window.
	release2, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("successor acquire: %v", err)
	}
	defer release2()

	successor, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the successor's record must exist: %v", err)
	}

	release1()

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the overlapped holder's release removed the successor's record: %v", err)
	}
	if !bytes.Equal(successor, after) {
		t.Fatalf("the successor's record changed across the overlapped release: %q became %q", successor, after)
	}
}

// TestAcquireGivingUpNamesTheHolderItWaitedBehind pins the give-up error's
// diagnosability: it wraps the context error and reports how long the
// caller queued and whom it was queued behind, so a bounded caller can
// print its failure and an operator knows which process to look at without
// opening the store directory.
func TestAcquireGivingUpNamesTheHolderItWaitedBehind(t *testing.T) {
	store := newStore(t, filelock.Options{})

	release, _, err := store.Acquire(context.Background(), testSubject)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, _, err = store.Acquire(ctx, testSubject)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("give-up error = %v, want one wrapping context.DeadlineExceeded", err)
	}

	msg := err.Error()
	if !strings.Contains(msg, "after ") || !strings.Contains(msg, "behind ") {
		t.Fatalf("give-up error %q must report how long it waited and behind whom", msg)
	}
	if !strings.Contains(msg, testPurpose) || !strings.Contains(msg, strconv.Itoa(os.Getpid())) {
		t.Fatalf("give-up error %q must name the holder it waited behind (pid %d, %q)", msg, os.Getpid(), testPurpose)
	}
}

// readmeContract and docContract extract the contract bullets from the two
// places the package states them, normalised to bare sentences, so
// TestContractIsTheSameTextInREADMEAndDoc can hold the README to the claim
// it makes about itself.
func readmeContract(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	var bullets []string
	in := false

	for l := range strings.SplitSeq(string(raw), "\n") {
		switch {
		case l == "## Contract":
			in = true
		case in && (strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "### ")):
			in = false
		case in && strings.HasPrefix(l, "- "):
			bullets = append(bullets, strings.TrimPrefix(l, "- "))
		case in && strings.HasPrefix(l, "  ") && len(bullets) > 0:
			bullets[len(bullets)-1] += " " + strings.TrimSpace(l)
		}
	}

	return normalizeBullets(bullets)
}

func docContract(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read doc.go: %v", err)
	}

	var bullets []string
	in := false

	for l := range strings.SplitSeq(string(raw), "\n") {
		switch {
		case l == "// # The contract":
			in = true
		case in && strings.HasPrefix(l, "// # "):
			in = false
		case in && strings.HasPrefix(l, "//   - "):
			bullets = append(bullets, strings.TrimPrefix(l, "//   - "))
		case in && strings.HasPrefix(l, "//     ") && len(bullets) > 0:
			bullets[len(bullets)-1] += " " + strings.TrimSpace(strings.TrimPrefix(l, "//"))
		}
	}

	return normalizeBullets(bullets)
}

// normalizeBullets strips the formatting that legitimately differs between
// markdown and a doc comment — bold markers, code backticks, wrapping — and
// leaves the words, which must not differ.
func normalizeBullets(in []string) []string {
	out := make([]string, 0, len(in))
	for _, b := range in {
		b = strings.ReplaceAll(b, "**", "")
		b = strings.ReplaceAll(b, "`", "")
		out = append(out, strings.Join(strings.Fields(b), " "))
	}

	return out
}

// TestContractIsTheSameTextInREADMEAndDoc keeps the README's promise about
// itself true mechanically: the Contract section of README.md and the
// contract section of doc.go are the same text, bullet for bullet, so a
// promise edited in one place cannot quietly diverge in the other.
func TestContractIsTheSameTextInREADMEAndDoc(t *testing.T) {
	readme := readmeContract(t)
	doc := docContract(t)

	if len(readme) == 0 || len(doc) == 0 {
		t.Fatalf("extracted %d README bullets and %d doc.go bullets — the extractors must find both contract sections", len(readme), len(doc))
	}

	if !slices.Equal(readme, doc) {
		limit := min(len(readme), len(doc))
		for i := range limit {
			if readme[i] != doc[i] {
				t.Fatalf("contract bullet %d differs:\n  README: %s\n  doc.go: %s", i+1, readme[i], doc[i])
			}
		}
		t.Fatalf("contract bullet count differs: %d in README.md, %d in doc.go", len(readme), len(doc))
	}
}
