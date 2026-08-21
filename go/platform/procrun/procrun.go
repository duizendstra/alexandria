package procrun

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotOnPath reports that a command name could not be resolved against the
// Runner's fixed PATH.
var ErrNotOnPath = errors.New("command not found on the fixed PATH")

// ErrNoOutput reports a Call passed to Run without an output file.
var ErrNoOutput = errors.New("Call.Output is required")

// DefaultTailLines is how many trailing output lines an ExitError carries when
// Runner.TailLines is zero.
const DefaultTailLines = 20

// outputPerm keeps a command's output readable by its owner only.
const outputPerm = 0o600

// Runner runs external commands under an environment it fully controls.
//
// A tool that shells out to other binaries usually cares about three things
// that os/exec does not do by itself: that no inherited variable can redirect
// the child (credentials, project selection, locale), that the binary actually
// executed is the one on a known PATH rather than whatever the parent happened
// to inherit, and that output lands in a file instead of a pipe that a reader
// may close early.
//
// The zero Runner inherits the parent environment unchanged and resolves
// commands with the parent's PATH — set the fields to tighten that.
type Runner struct {
	// Path is the PATH handed to the child. When set, a Call name without a
	// path separator is resolved against it rather than against the parent's
	// PATH, so an inherited PATH cannot decide which binary runs.
	Path string

	// Env are variables set on every call, after scrubbing. They override
	// inherited values of the same name.
	Env map[string]string

	// Scrub drops every inherited variable whose name carries one of these
	// prefixes. Use it for families that redirect a child's identity or
	// configuration.
	Scrub []string

	// TailLines is how many trailing output lines an ExitError carries.
	// Zero means DefaultTailLines; a negative value means none.
	TailLines int
}

// Call is one command to run.
type Call struct {
	// Name is the binary. Without a path separator it is resolved against the
	// Runner's Path.
	Name string
	Args []string

	// Env are variables for this call only, applied last. Use it for a value
	// that must reach one command and no other.
	Env map[string]string

	// Output is the file that receives stdout and stderr. Required for Run.
	Output string

	// Append adds to an existing Output file instead of truncating it.
	Append bool
}

// ExitError reports a command that ran and finished with a non-zero code.
// A command that could not start at all returns a different error.
type ExitError struct {
	Name   string
	Code   int
	Output string // path of the output file, when there was one.
	Tail   string // the last lines of that file.
}

func (e *ExitError) Error() string {
	s := fmt.Sprintf("%s exited with code %d", e.Name, e.Code)
	if e.Output != "" {
		s += " — output: " + e.Output
	}

	return s
}

// ExitCodeOf returns the exit code carried by err, or -1 when err is nil or
// carries none.
func ExitCodeOf(err error) int {
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}

	return -1
}

// Run executes the call and writes stdout and stderr to Call.Output.
//
// Output goes to a file rather than a pipe on purpose: a reader that stops
// early closes the pipe and the child dies on SIGPIPE, which turns a complete
// result into a truncated one without an error to show for it.
func (r *Runner) Run(ctx context.Context, c *Call) error {
	if c.Output == "" {
		return fmt.Errorf("%s: %w", c.Name, ErrNoOutput)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if c.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(c.Output, flags, outputPerm)
	if err != nil {
		return fmt.Errorf("output file %s: %w", c.Output, err)
	}
	defer func() { _ = f.Close() }()

	cmd, err := r.command(ctx, c)
	if err != nil {
		return err
	}

	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Run(); err != nil {
		return r.exitError(c, err)
	}

	return nil
}

// Capture executes the call and returns its standard output.
//
// Use it for short, fully-read output such as a version banner; Run is the
// right call for anything whose output belongs in a log.
func (r *Runner) Capture(ctx context.Context, c *Call) (string, error) {
	cmd, err := r.command(ctx, c)
	if err != nil {
		return "", err
	}

	out, err := cmd.Output()
	if err != nil {
		return string(out), r.exitError(c, err)
	}

	return string(out), nil
}

// Environ composes the environment for a call: the inherited environment minus
// every scrubbed prefix and every name the Runner or the call sets, then the
// Runner's own variables, then the call's. It is exported so callers can assert
// on the exact environment a command would receive.
func (r *Runner) Environ(callEnv map[string]string) []string {
	fixed := make(map[string]string, len(r.Env)+len(callEnv)+1)
	maps.Copy(fixed, r.Env)

	maps.Copy(fixed, callEnv)

	if r.Path != "" {
		fixed["PATH"] = r.Path
	}

	env := make([]string, 0, len(os.Environ())+len(fixed))

	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if _, replaced := fixed[k]; replaced || r.scrubbed(k) {
			continue
		}

		env = append(env, kv)
	}

	for k, v := range fixed {
		env = append(env, k+"="+v)
	}

	return env
}

// LookPath resolves a command name against the Runner's fixed Path. A name
// that already carries a path separator is returned unchanged, and an empty
// Path falls back to the parent's PATH.
func (r *Runner) LookPath(name string) (string, error) {
	if strings.ContainsRune(name, os.PathSeparator) {
		return name, nil
	}

	if r.Path == "" {
		found, err := exec.LookPath(name)
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}

		return found, nil
	}

	for _, dir := range filepath.SplitList(r.Path) {
		if dir == "" {
			continue
		}

		candidate := filepath.Join(dir, name)
		if executable(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%s: %w (%s)", name, ErrNotOnPath, r.Path)
}

func executable(path string) bool {
	fi, err := os.Stat(path)

	return err == nil && !fi.IsDir() && fi.Mode().Perm()&0o111 != 0
}

// tail returns the last n lines of a file, for an error message.
func tail(path string, n int) string {
	if path == "" || n <= 0 {
		return ""
	}

	b, err := os.ReadFile(path) //nolint:gosec // the caller owns this path
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return strings.Join(lines, "\n")
}

// command builds the exec.Cmd with the resolved binary and the composed
// environment.
func (r *Runner) command(ctx context.Context, c *Call) (*exec.Cmd, error) {
	name, err := r.LookPath(c.Name)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, name, c.Args...) //nolint:gosec // running named binaries is this package's purpose
	cmd.Env = r.Environ(c.Env)
	cmd.Stdin = nil

	return cmd, nil
}

// exitError converts an exec failure into an ExitError, or passes through the
// error of a command that never started.
func (r *Runner) exitError(c *Call, err error) error {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return fmt.Errorf("%s could not run: %w", c.Name, err)
	}

	return &ExitError{
		Name:   c.Name,
		Code:   ee.ExitCode(),
		Output: c.Output,
		Tail:   tail(c.Output, r.tailLines()),
	}
}

func (r *Runner) tailLines() int {
	if r.TailLines == 0 {
		return DefaultTailLines
	}

	return r.TailLines
}

func (r *Runner) scrubbed(name string) bool {
	for _, p := range r.Scrub {
		if strings.HasPrefix(name, p) {
			return true
		}
	}

	return false
}
