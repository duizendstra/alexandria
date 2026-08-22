//go:build windows

package runstate_test

import (
	"syscall"
	"testing"
)

// syscallSIGUSR1 has no Windows equivalent; SIGINT keeps the test compiling.
// The signal-raising test skips on Windows (see raise).
const syscallSIGUSR1 = syscall.SIGINT

func raise(t *testing.T, _ syscall.Signal) {
	t.Helper()
	t.Skip("self-signal raising is not supported on windows")
}
