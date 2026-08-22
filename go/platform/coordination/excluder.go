package coordination

import (
	"errors"
)

// ErrLocked reports that another worker or process already holds the exclusion lock.
var ErrLocked = errors.New("exclusion lock already held")

// Excluder provides mutual exclusion for a named subject.
type Excluder interface {
	// Acquire attempts to acquire exclusive access for the subject.
	// On success, it returns a release function that frees the lock (safe to call repeatedly)
	// and a nil error.
	// If the lock is already held, it returns an error wrapping or matching ErrLocked.
	Acquire(subject string) (func(), error)
}

// NopExcluder returns an Excluder that always succeeds and performs no-op releases.
// Useful for hermetic unit testing or single-threaded execution modes.
func NopExcluder() Excluder { //nolint:ireturn // factory returns interface by design
	return nopExcluder{}
}

type nopExcluder struct{}

func (nopExcluder) Acquire(string) (func(), error) {
	return func() {}, nil
}
