//go:build windows

package runstate

import "os"

// interruptExitCode is the process exit code used after an interrupt is handled
// on Windows, which has no 128+signal convention.
const interruptExitCode = 1

// reraise terminates the process after the lock file has been removed. Windows
// has no self-signal facility (syscall.Kill is undefined there), and
// signal.Notify has already diverted the interrupt from the default handler, so
// without an explicit exit the process would keep running after a Ctrl-C.
func reraise(_ os.Signal) {
	os.Exit(interruptExitCode)
}
