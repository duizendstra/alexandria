package procrun_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/duizendstra/alexandria/go/platform/procrun"
)

// A Runner gives a child process exactly the environment it should have: a
// fixed PATH, fixed variables, and no inherited member of a scrubbed family.
func ExampleRunner_Run() {
	dir, _ := os.MkdirTemp("", "procrun")
	defer func() { _ = os.RemoveAll(dir) }()

	log := filepath.Join(dir, "tool.log")

	r := procrun.Runner{
		Path:  fixedPath,
		Env:   map[string]string{envLocale: "C"},
		Scrub: []string{vendorPrefix},
	}

	err := r.Run(context.Background(), &procrun.Call{
		Name:   "sh",
		Args:   []string{"-c", "echo hello; echo trouble >&2"},
		Output: log,
	})
	if err != nil {
		fmt.Println("failed:", err)

		return
	}

	// Both streams landed in the file, in order.
	b, _ := os.ReadFile(log)
	fmt.Print(string(b))
	// Output:
	// hello
	// trouble
}

// A variable that must reach one command and no other belongs on the Call.
func ExampleRunner_Capture() {
	r := procrun.Runner{Path: fixedPath, Scrub: []string{vendorPrefix}}

	out, err := r.Capture(context.Background(), &procrun.Call{
		Name: "sh",
		Args: []string{"-c", `echo "identity=$VENDOR_IDENTITY"`},
		Env:  map[string]string{envIdentity: identityReader},
	})
	if err != nil {
		fmt.Println("failed:", err)

		return
	}

	fmt.Print(out)
	// Output: identity=reader
}

// A non-zero exit is an *ExitError carrying the code and the tail of the
// output, so the caller can map it onto its own exit classes.
func ExampleExitCodeOf() {
	dir, _ := os.MkdirTemp("", "procrun")
	defer func() { _ = os.RemoveAll(dir) }()

	r := procrun.Runner{Path: fixedPath}

	err := r.Run(context.Background(), &procrun.Call{
		Name:   "sh",
		Args:   []string{"-c", "echo refused >&2; exit 7"},
		Output: filepath.Join(dir, "tool.log"),
	})

	fmt.Println("exit code:", procrun.ExitCodeOf(err))
	// Output: exit code: 7
}
