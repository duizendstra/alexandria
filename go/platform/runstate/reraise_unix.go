//go:build !windows

package runstate

import (
	"os"
	"syscall"
)

// reraise re-delivers a caught signal to this process after the lock file has
// been removed, so the process exits with the conventional 128+n code instead
// of having the signal swallowed by our own handler.
func reraise(s os.Signal) {
	if nr, ok := s.(syscall.Signal); ok {
		_ = syscall.Kill(os.Getpid(), nr)
	}
}
