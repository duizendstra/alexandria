package filelock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

// dirPerm and filePerm keep the store readable by its owner only.
const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// LockSuffix and FenceSuffix name the two files one subject owns inside the
// store: <Dir>/<subject><LockSuffix> exists exactly while the window is
// held, and <Dir>/<subject><FenceSuffix> holds that subject's counter and
// outlives every individual occupancy.
const (
	LockSuffix  = ".lock"
	FenceSuffix = ".fence"
)

// ReclaimedSuffix marks a live holder record a reclaimer set aside and could
// not put back: <Dir>/<subject><LockSuffix><ReclaimedSuffix>.<random>. Such a
// file exists for microseconds during any reclaim, and it persists only as
// evidence of the one raced outcome takeAside cannot undo — see there.
// Removing a persistent one discards that evidence, nothing else: it claims
// no window.
const ReclaimedSuffix = ".reclaimed"

// DefaultPollMin and DefaultPollMax bound the jittered wait between attempts
// while somebody else holds the window. Short enough that a window held
// around a handful of calls is picked up promptly; jittered so two waiters
// do not retry in lockstep and collide again on every attempt.
const (
	DefaultPollMin = 100 * time.Millisecond
	DefaultPollMax = 400 * time.Millisecond
)

// Options tunes a [Store]. Every zero value is a deliberate default: the
// polling bounds fall back to [DefaultPollMin]/[DefaultPollMax], and
// StaleAfter zero means reclaim is OFF.
type Options struct {
	// PollMin and PollMax bound the jittered wait between attempts. Both
	// zero means the defaults; PollMax must exceed PollMin.
	PollMin time.Duration
	PollMax time.Duration

	// StaleAfter is the age past which a holder record is treated as left
	// behind and reclaimed. Zero — the zero value, and the recommended
	// default — means reclaim is off: a record is never removed on another
	// caller's behalf, whatever its age.
	//
	// This is a safety-versus-liveness trade and there is no value that is
	// right for every caller, so there is deliberately no built-in
	// non-zero default: a caller that wants reclaim states the age at its
	// own composition root, where it can be reviewed. See the package
	// documentation.
	StaleAfter time.Duration
}

// Store is a [coordination.Waiter] over one directory. Two processes that
// name the same Dir and the same subject queue for one window.
//
// Dir is created if missing; the caller owns it and nothing else may be
// written there. Purpose is recorded in every holder record this Store
// writes — a short phrase naming the work, for whoever reads the record
// later. Options are read on every Acquire, so a Store is safe to share.
//
// A Store makes no distributed-locking claim: see the package
// documentation for what it does and does not guarantee.
type Store struct {
	Dir     string
	Purpose string
	Options Options
}

// Store satisfies the Waiter half of the published language; the compiler
// keeps that true.
var _ coordination.Waiter = (*Store)(nil)

// LockPath is where the holder record for subject lives while the window is
// held. It exists for operators and for tests; Acquire does not need it.
func (s *Store) LockPath(subject coordination.Subject) (string, error) {
	return s.path(subject, LockSuffix)
}

// FencePath is where subject's counter lives. Unlike the holder record it
// is not removed on release: the counter has to survive so the next
// acquisition can advance it.
func (s *Store) FencePath(subject coordination.Subject) (string, error) {
	return s.path(subject, FenceSuffix)
}

// Holder reads the record of whoever is inside subject's window. The second
// result is false when nobody holds it. A record that exists but cannot be
// parsed is reported as held by a zero-age holder rather than as an error:
// something is in there, and refusing to say so would be worse than saying
// so vaguely.
func (s *Store) Holder(subject coordination.Subject) (coordination.Holder, bool, error) {
	path, err := s.LockPath(subject)
	if err != nil {
		return coordination.Holder{}, false, err
	}

	h, _, ok := readHolder(path)
	if !ok {
		return coordination.Holder{}, false, nil
	}

	return h, true, nil
}

// Acquire enters subject's window, queueing behind whoever is inside it,
// until ctx is done.
//
// On success it returns a release that leaves the window — idempotent, safe
// to defer unconditionally — and the fence this occupancy advanced the
// subject's counter to. The counter is read, incremented and rewritten
// while the window is held, so two occupancies never share a value and a
// later one always carries a higher one.
//
// A window claimed by a record older than Options.StaleAfter is reclaimed
// when StaleAfter is set, and waited on forever otherwise. A record that is
// found abandoned but cannot be cleared ends the wait with an error
// wrapping [coordination.ErrStaleLock] rather than looping in silence.
//
// A wait that ends on ctx instead reports how long it queued and whom it was
// queued behind, so a bounded caller's give-up is diagnosable from the error
// alone.
//
//nolint:gocritic // unnamedResult: the result shape is the coordination.Waiter contract, and nonamedreturns forbids naming it here.
func (s *Store) Acquire(ctx context.Context, subject coordination.Subject) (func(), uint64, error) {
	path, err := s.LockPath(subject)
	if err != nil {
		return nil, 0, err
	}

	if err := os.MkdirAll(s.Dir, dirPerm); err != nil {
		return nil, 0, fmt.Errorf("coordination store %s: %w", s.Dir, err)
	}

	started := time.Now()

	for {
		mine, claimErr := s.claim(path)
		if claimErr == nil {
			return s.enter(path, subject, mine)
		}

		if !errors.Is(claimErr, os.ErrExist) {
			return nil, 0, claimErr
		}

		if err := s.reclaim(path); err != nil {
			return nil, 0, err
		}

		select {
		case <-time.After(s.jitteredPoll()):
		case <-ctx.Done():
			return nil, 0, fmt.Errorf("enter %s: %w after %s behind %s",
				path, ctx.Err(), time.Since(started).Round(time.Millisecond), occupant(path))
		}
	}
}

// occupant describes whoever holds the record at path, for a wait that is
// about to give up. A wait can end exactly between a release and the next
// attempt, so "nobody" is a real answer; so is a record that cannot be
// parsed, which is still a claim (see readHolder) but cannot be described.
func occupant(path string) string {
	h, _, ok := readHolder(path)

	switch {
	case !ok:
		return "nobody (the window was free at the last look)"
	case h.PID == 0 && h.Host == "":
		return "an unreadable record (present, but not parseable as a holder)"
	default:
		return h.String()
	}
}

// path composes <Dir>/<subject><suffix> after validating the subject.
func (s *Store) path(subject coordination.Subject, suffix string) (string, error) {
	if err := subject.Validate(); err != nil {
		return "", fmt.Errorf("%w", err)
	}

	return filepath.Join(s.Dir, string(subject)+suffix), nil
}

// claim creates the holder record at path in one indivisible step, or
// reports that somebody else is already inside the window.
//
// The complete record is written to a temporary file in the same directory
// and fsynced there, and only then linked into place. A hard link refuses to
// replace an existing target, so the link is at once the atomic creation of
// the record and the exclusivity this lock is entirely made of — and the
// record is never visible in a half-written state, because the name only
// ever appears attached to content that was already complete on disk. A
// process killed anywhere in here leaves at most a stray temporary file,
// which claims nothing.
//
// An existing target is reported as an error matching [os.ErrExist] (via
// errors.Is), because that is the answer the caller's wait is built on. On
// success, claim also returns the published record's file identity — what
// releaseOwn later uses to make sure a release removes its OWN record and
// never a successor's.
//
// The record is composed HERE, once per attempt, and deliberately not once
// per wait. It carries the instant the window was entered, which is what
// decides later whether its holder looks abandoned. Composing it once and
// reusing it across a long queue would save a create and an fsync per poll
// and would stamp the record at the moment the caller started waiting: a
// caller that queued for longer than a reclaim age would then enter a window
// whose record is already old enough for the next caller to take away from
// it, with two callers inside and nothing to report it. The per-attempt
// staging file is the price of the stamp being true.
//
// The directory is fsynced after the link for the same reason [writeAtomic]
// fsyncs it after its rename: otherwise a host that dies just after the link
// can come back with the directory entry missing. A claim that cannot be
// made durable is undone and failed rather than handed out — a caller whose
// claim might not survive the next second was never inside.
func (s *Store) claim(path string) (os.FileInfo, error) {
	dir := filepath.Dir(path)

	record, err := json.Marshal(coordination.Self(s.Purpose))
	if err != nil {
		return nil, fmt.Errorf("enter %s: holder record: %w", path, err)
	}

	tmp, err := writeTemp(dir, filepath.Base(path), append(record, '\n'))
	if err != nil {
		return nil, fmt.Errorf("enter %s: %w", path, err)
	}

	// The record's identity is fixed here, before the link: linking adds a
	// second name for this same file, so the staging file's identity IS the
	// published record's, with no moment in which reading it could race.
	mine, err := os.Stat(tmp)
	if err != nil {
		_ = os.Remove(tmp)

		return nil, fmt.Errorf("enter %s: %w", path, err)
	}

	// The temporary file has done its job either way: it is the record when
	// the link lands, and it is rubbish when the link is refused. Removing it
	// on both branches is what keeps a long wait from leaving one behind per
	// attempt.
	linkErr := os.Link(tmp, path)
	_ = os.Remove(tmp)

	if linkErr != nil {
		return nil, fmt.Errorf("enter %s: %w", path, linkErr)
	}

	if err := fsyncDir(dir); err != nil {
		releaseOwn(path, mine)

		return nil, fmt.Errorf("enter %s: %w", path, err)
	}

	return mine, nil
}

// enter finishes an acquisition whose holder record is already complete on
// disk: it advances the fence and hands back the release. A fence that
// cannot be advanced fails the acquisition and leaves the window — a caller
// that got no fence was never inside.
//
// The release removes only the record this occupancy published (releaseOwn):
// a holder that was reclaimed while still inside must not take the NEW
// holder's record with it on the way out.
//
//nolint:gocritic // unnamedResult: enter returns exactly what Acquire promises, and nonamedreturns forbids naming it here.
func (s *Store) enter(path string, subject coordination.Subject, mine os.FileInfo) (func(), uint64, error) {
	fence, err := s.advanceFence(subject)
	if err != nil {
		releaseOwn(path, mine)

		return nil, 0, err
	}

	var once sync.Once

	return func() { once.Do(func() { releaseOwn(path, mine) }) }, fence, nil
}

// releaseOwn removes the holder record at path only if it is still the one
// this occupancy published — compared by file identity, never by content. A
// record that is gone, or that is somebody else's, is left alone: it means
// this holder was reclaimed while it was still inside (a reclaim age shorter
// than the occupancy turned out to be), and the overlap that caused is
// already done — the fence is what makes it visible downstream. Removing the
// new record too would not undo the overlap; it would hand the window to yet
// a THIRD caller while the new holder still believes it is inside.
//
// The check narrows the old unconditional remove to a read-then-remove: in
// the interval between the two, only a reclaim can take the record away, and
// only once this record is itself past the staleness age — at which point
// removing the successor is exactly what this function exists to prevent.
func releaseOwn(path string, mine os.FileInfo) {
	cur, err := os.Stat(path)
	if err != nil || !os.SameFile(cur, mine) {
		return
	}

	_ = os.Remove(path)
}

// advanceFence reads subject's counter, increments it and rewrites it
// atomically. It runs only while the window is held, so the read-then-write
// cannot race another holder of the same subject.
func (s *Store) advanceFence(subject coordination.Subject) (uint64, error) {
	path, err := s.FencePath(subject)
	if err != nil {
		return 0, err
	}

	next := uint64(1)

	switch b, readErr := os.ReadFile(path); { //nolint:gosec // the caller owns this directory
	case readErr == nil:
		n, parseErr := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("fence %s: %w", path, parseErr)
		}
		next = n + 1
	case !errors.Is(readErr, os.ErrNotExist):
		return 0, fmt.Errorf("fence %s: %w", path, readErr)
	}

	if err := writeAtomic(path, []byte(strconv.FormatUint(next, 10)+"\n")); err != nil {
		return 0, fmt.Errorf("fence %s: %w", path, err)
	}

	return next, nil
}

// reclaim clears a holder record older than Options.StaleAfter so the wait
// can end. With StaleAfter unset it does nothing at all: reclaim is opt-in.
//
// A record that has disappeared in the meantime, or that changed hands
// before it could be set aside, is not an error — the next attempt in
// Acquire's loop is the real answer either way. A record that is abandoned
// by the configured measure and still cannot be cleared IS an error: that
// wait would never end.
func (s *Store) reclaim(path string) error {
	if s.Options.StaleAfter <= 0 {
		return nil
	}

	h, judged, ok := readHolder(path)
	if !ok || h.Age(time.Now().UTC()) <= s.Options.StaleAfter {
		return nil
	}

	return takeAside(path, h, judged)
}

// takeAside removes the record judged abandoned — and ONLY that record. An
// unconditional remove-by-path here would race the judged holder's own
// release: the slow holder releases, a fresh caller claims, and the remove
// deletes the FRESH record — two callers inside, with nothing abandoned
// anywhere. Two reclaimers behind one dead holder race the same way. So the
// record is set ASIDE by rename, onto a name only this call uses, and
// identity is checked on what actually moved:
//
//   - It is the judged record: reclaimed. The set-aside copy is rubbish and
//     is removed.
//   - It is somebody else's — a live claim replaced the judged record
//     between the read and the rename: it is put straight back. The restore
//     links rather than renames, so it refuses to clobber yet another claim,
//     and the caller goes back to waiting.
//
// The gap in which a live record is missing from its path is rename-to-link,
// two directory operations with no fsync between them. If a third caller
// claims in exactly that gap the restore is refused: the moved record is
// preserved under [ReclaimedSuffix] as evidence, and the error reports that
// two callers are now inside — the one raced outcome a filesystem without a
// conditional remove cannot rule out, and the fence is what keeps even that
// one distinguishable downstream.
func takeAside(path string, judged coordination.Holder, judgedFi os.FileInfo) error {
	tomb, err := reserveTomb(path)
	if err != nil {
		return fmt.Errorf("reclaim %s held by %s: %w: %w", path, judged, coordination.ErrStaleLock, err)
	}

	if err := os.Rename(path, tomb); err != nil {
		_ = os.Remove(tomb)

		if errors.Is(err, os.ErrNotExist) {
			return nil // Released, or reclaimed by somebody else, in the meantime.
		}

		return fmt.Errorf("reclaim %s held by %s: %w: %w", path, judged, coordination.ErrStaleLock, err)
	}

	moved, err := os.Stat(tomb)
	if err != nil {
		return fmt.Errorf("reclaim %s held by %s: record set aside at %s: %w", path, judged, tomb, err)
	}

	if os.SameFile(moved, judgedFi) {
		_ = os.Remove(tomb)

		return nil
	}

	if err := os.Link(tomb, path); err != nil {
		return fmt.Errorf(
			"reclaim %s: set aside a LIVE record (%s) and another claim landed before it could be restored"+
				" — two callers are inside this window and the fence is the only witness; the moved record is preserved at %s: %w",
			path, occupant(tomb), tomb, err)
	}

	_ = os.Remove(tomb)

	return nil
}

// reserveTomb creates the empty, uniquely named file a takeAside will rename
// the judged record onto. Uniqueness comes from the OS: two reclaimers of
// one subject must never set records aside onto the same name, or one
// reclaimer's cleanup would delete the record the other had just moved.
func reserveTomb(path string) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+ReclaimedSuffix+".*")
	if err != nil {
		return "", fmt.Errorf("reserve set-aside name: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())

		return "", fmt.Errorf("reserve set-aside name %s: %w", f.Name(), err)
	}

	return f.Name(), nil
}

// readHolder reads the record at path, together with the file identity of
// exactly what it read — the FileInfo comes from the open handle, so it can
// never describe a different record than the bytes do. The last result is
// false when there is no record (or it vanished mid-read). A record that
// does not parse yields a zero holder (age 0, never reclaimable) rather
// than an error: an unreadable claim is still a claim, and reclaiming what
// cannot be read would be the more dangerous reading.
//
// Since is defaulted to the file's modification time when the record parses
// but carries no Since of its own, so a record written by an older writer
// still ages.
func readHolder(path string) (coordination.Holder, os.FileInfo, bool) {
	f, err := os.Open(path) //nolint:gosec // the caller owns this directory
	if err != nil {
		return coordination.Holder{}, nil, false
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return coordination.Holder{}, nil, false
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return coordination.Holder{}, nil, false
	}

	var h coordination.Holder
	if err := json.Unmarshal(b, &h); err != nil {
		return coordination.Holder{}, fi, true
	}

	if h.Since.IsZero() {
		h.Since = fi.ModTime().UTC()
	}

	return h, fi, true
}

// writeTemp writes content to a fresh temporary file next to the name it is
// destined for, and returns that temporary file's path for the caller to
// publish. It is the half [writeAtomic] and [Store.claim] share: the content
// is fsynced before the file is closed, so it is on disk ahead of whatever
// rename or link makes it visible under its real name, and the mode is
// narrowed to filePerm because this temporary file becomes the published
// file itself rather than being copied into it. Nothing is left behind on
// failure — the caller's directory is as it was found.
//
// What it deliberately does NOT do is publish: renaming replaces an existing
// target and linking refuses one, and which of those two a caller needs is
// the whole difference between a counter and a claim.
func writeTemp(dir, prefix string, content []byte) (string, error) {
	f, err := os.CreateTemp(dir, prefix+".*")
	if err != nil {
		return "", fmt.Errorf("temp file in %s: %w", dir, err)
	}
	tmp := f.Name()

	_, err = f.Write(content)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp, filePerm)
	}

	if err != nil {
		_ = os.Remove(tmp)

		return "", fmt.Errorf("temp file %s: %w", tmp, err)
	}

	return tmp, nil
}

// writeAtomic replaces path with content: a temporary file in the same
// directory, fsynced, then a rename, so a reader never sees a half-written
// counter. The directory is fsynced after the rename so the rename itself
// survives an unclean crash — otherwise a host that dies right after the
// rename can come back up with the directory entry still pointing at the old
// inode, and the counter reverts to a value it already handed out. Content
// durability is unconditional; the directory fsync is a POSIX facility (see
// [fsyncDir]) and is best effort where the platform does not support it, but
// a failure it does report is not swallowed.
//
// Rename REPLACES its target, which is right for a counter that is meant to
// be overwritten and wrong for anything whose existence is a claim: see
// [Store.claim], which links instead precisely because linking refuses.
func writeAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)

	tmp, err := writeTemp(dir, filepath.Base(path), content)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	err = os.Rename(tmp, path)
	if err == nil {
		err = fsyncDir(dir)
	}

	if err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// fsyncDir fsyncs the directory entry itself, which is what makes a rename
// durable across an unclean crash — fsyncing the file alone only guarantees
// its content, not that the directory's pointer to it survives. Directory
// fsync is a POSIX facility: opening a directory for [os.File.Sync] is not
// supported on Windows, so there this is a deliberate, silent no-op rather
// than a failure of the write. Where it is supported, a failure is returned,
// not swallowed.
func fsyncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	d, err := os.Open(dir) //nolint:gosec // the caller owns this directory
	if err != nil {
		return fmt.Errorf("open dir %s: %w", dir, err)
	}

	err = d.Sync()
	if closeErr := d.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return fmt.Errorf("fsync dir %s: %w", dir, err)
	}

	return nil
}

// jitteredPoll returns a random duration in [PollMin, PollMax), falling back
// to the package defaults when the Options leave them unset or inverted.
func (s *Store) jitteredPoll() time.Duration {
	minWait, maxWait := s.Options.PollMin, s.Options.PollMax
	if minWait <= 0 || maxWait <= minWait {
		minWait, maxWait = DefaultPollMin, DefaultPollMax
	}

	return minWait + time.Duration(rand.Int64N(int64(maxWait-minWait))) //nolint:gosec // G404: poll-interval jitter, not a security value.
}
