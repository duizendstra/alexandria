//go:build !windows

package runstate_test

import (
	"os"
	"syscall"
	"testing"
)

// syscallSIGUSR1 is a signal that is safe to raise in a test: nothing else in
// the test binary listens for it.
const syscallSIGUSR1 = syscall.SIGUSR1

func raise(t *testing.T, s syscall.Signal) {
	t.Helper()

	if err := syscall.Kill(os.Getpid(), s); err != nil {
		t.Fatalf("raise %v: %v", s, err)
	}
}
