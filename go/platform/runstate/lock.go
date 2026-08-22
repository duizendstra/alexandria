package runstate

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

// ErrLocked reports that another run already holds the lock for a subject.
var ErrLocked = errors.New("another run already holds this lock")

// DefaultLockPrefix is the file-name prefix a Locker uses when Prefix is empty.
const DefaultLockPrefix = "lock."

// Locker gives mutual exclusion per subject, using a file created atomically
// with O_CREATE|O_EXCL. The lock is released when the returned function is
// called, and also when the process is interrupted — a lock left behind by a
// Ctrl-C would block the next run for no reason.
type Locker struct {
	// Dir holds the lock files. It is created if missing.
	Dir string

	// Prefix goes in front of the subject in the file name.
	// Empty means DefaultLockPrefix.
	Prefix string

	// Signals are the signals that trigger cleanup. Nil means SIGINT and
	// SIGTERM.
	Signals []os.Signal

	// OnSignal runs after the lock is cleaned up, instead of re-raising the
	// signal at the process. Set it in tests, or when the caller wants to
	// shut down in its own way.
	OnSignal func(os.Signal)
}

// Compile-time interface assertion.
var _ coordination.Excluder = (*Locker)(nil)

// Acquire takes the lock for a subject. The returned release is safe to call
// more than once. It returns ErrLocked when the lock is already held.
func (l *Locker) Acquire(subject string) (func(), error) {
	path, err := pathFor(l.Dir, l.prefix(), subject, "")
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(l.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("lock directory %s: %w", l.Dir, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm) //nolint:gosec // the caller owns this directory
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%s (%s): %w", subject, path, ErrLocked)
		}

		return nil, fmt.Errorf("lock %s: %w", path, err)
	}

	_, _ = fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	_ = f.Close()

	var removeOnce sync.Once

	remove := func() { removeOnce.Do(func() { _ = os.Remove(path) }) }

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, l.signals()...)

	go func() {
		s, ok := <-sig
		if !ok {
			return
		}

		remove()

		if l.OnSignal != nil {
			l.OnSignal(s)

			return
		}

		signal.Stop(sig)

		// Re-raise, so the shell still sees the usual 128+n exit code.
		if nr, ok := s.(syscall.Signal); ok {
			_ = syscall.Kill(os.Getpid(), nr)
		}
	}()

	var stopOnce sync.Once

	return func() {
		stopOnce.Do(func() {
			signal.Stop(sig)
			close(sig)
		})
		remove()
	}, nil
}

func (l *Locker) prefix() string {
	if l.Prefix == "" {
		return DefaultLockPrefix
	}

	return l.Prefix
}

func (l *Locker) signals() []os.Signal {
	if len(l.Signals) == 0 {
		return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	return l.Signals
}
