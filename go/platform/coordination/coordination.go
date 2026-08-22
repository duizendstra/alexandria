package coordination

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrBadSubject reports a subject that cannot safely become part of a path.
var ErrBadSubject = errors.New("subject may not be empty, a parent reference, or contain a path separator")

// ErrLocked reports that another holder is already inside the window. It is
// an [Excluder]'s refusal; a [Waiter] queues instead of returning it.
var ErrLocked = errors.New("another holder is already inside this window")

// ErrStaleLock reports a holder record left behind by a holder that is gone
// — a hard kill, a crash, a machine that lost power. An adapter configured
// to reclaim removes such a record and proceeds; one that cannot remove it,
// or that is not configured to reclaim at all, reports this instead of
// looping in silence.
var ErrStaleLock = errors.New("the window is claimed by a holder record that was left behind")

// Subject names one window: the smallest stretch of work during which the
// shared resource may not be mutated by anybody else.
//
// A subject is chosen by the context that owns the resource, and it is
// opaque to whoever allocates the store the records live in. It becomes
// part of a path, so it is validated ([Subject.Validate]) before it is
// used, however trusted its origin.
type Subject string

// String returns the subject as a plain string.
func (s Subject) String() string { return string(s) }

// Validate reports whether the subject can safely become part of a path.
// The empty string, "." and "..", anything containing a path separator, and
// anything containing a parent reference are rejected with an error
// wrapping [ErrBadSubject].
func (s Subject) Validate() error {
	subject := string(s)
	switch {
	case subject == "", subject == "." || subject == "..":
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	case strings.ContainsRune(subject, os.PathSeparator), strings.ContainsRune(subject, '/'):
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	case strings.Contains(subject, ".."):
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	}

	return nil
}

// Holder is the record of who is inside a window. An adapter writes it when
// the window is entered and removes it on release; anybody may read it,
// which is the point — an operator looking at a window nobody seems to be
// using should be able to see, without any tooling, which process on which
// machine claimed it and when.
//
// Since is what an adapter compares against its reclaim age, so it is
// recorded in UTC and never rewritten while the window is held.
type Holder struct {
	PID     int       `json:"pid"`
	Host    string    `json:"host"`
	Since   time.Time `json:"since"`
	Purpose string    `json:"purpose"`
}

// Self describes the calling process as a [Holder], with Since set to the
// current UTC time. Purpose is a short, human-readable phrase naming the
// work the window is being held for — it is read by whoever finds the
// record, so write it for them.
func Self(purpose string) Holder {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	return Holder{PID: os.Getpid(), Host: host, Since: time.Now().UTC(), Purpose: purpose}
}

// String renders the holder as one operator-readable line.
func (h Holder) String() string {
	purpose := h.Purpose
	if purpose == "" {
		purpose = "unstated purpose"
	}

	return fmt.Sprintf("pid %d on %s since %s (%s)", h.PID, h.Host, h.Since.Format(time.RFC3339), purpose)
}

// Age reports how long the holder has been inside the window, as of now. A
// record whose Since is zero (unset, or unparseable and defaulted) has no
// meaningful age and reports 0, so an adapter comparing against a reclaim
// age never treats an unreadable record as abandoned by accident.
func (h Holder) Age(now time.Time) time.Duration {
	if h.Since.IsZero() {
		return 0
	}

	return now.Sub(h.Since)
}

// Waiter enters a window, queueing behind whoever is already inside it.
//
// Acquire blocks until the window is entered or ctx is done, whichever
// comes first. It returns a release that leaves the window — idempotent,
// safe to defer unconditionally — and a fence: a per-subject counter,
// strictly increasing, advanced while the window is held, for recording
// alongside whatever the window guarded. A failed Acquire returns a nil
// release, a zero fence and an error; ctx's error is wrapped when the wait
// was given up on.
//
// Use a Waiter when both callers have legitimate work and the only problem
// is simultaneity.
type Waiter interface {
	Acquire(ctx context.Context, subject Subject) (release func(), fence uint64, err error)
}

// Excluder enters a window or refuses.
//
// TryAcquire returns an error wrapping [ErrLocked] when another holder is
// already inside; it never waits. The returned release is idempotent and
// safe to defer unconditionally.
//
// Use an Excluder when a second concurrent attempt is a mistake in itself,
// and the operator is better served by hearing about it now than by having
// it silently queued.
type Excluder interface {
	TryAcquire(subject Subject) (release func(), err error)
}
