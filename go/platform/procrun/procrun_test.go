package procrun_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/procrun"
)

// Shared literals for these tests; goconst wants them named once.
const (
	fixedPath      = "/usr/bin:/bin"
	shellPath      = "/bin/sh"
	envLocale      = "LC_ALL"
	envIdentity    = "VENDOR_IDENTITY"
	vendorPrefix   = "VENDOR_"
	identityReader = "reader"
)

// stub writes a shell script that reports its arguments, its environment and
// whether its stdout is a pipe, then exits with the given code.
func stub(t *testing.T, dir, name string, exitCode int) string {
	t.Helper()

	path := filepath.Join(dir, name)
	script := `#!/bin/sh
echo "ARGS: $*"
if [ -p /dev/stdout ]; then echo "STDOUT: pipe"; else echo "STDOUT: file"; fi
env | sed 's/^/ENV /'
echo "on stderr" >&2
exit ` + strconv.Itoa(exitCode) + `
`

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	return path
}

func read(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}

func TestRunWritesBothStreamsToAFile(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)
	out := filepath.Join(dir, "run.log")

	var r procrun.Runner
	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Args: []string{"one", "two"}, Output: out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := read(t, out)
	for _, want := range []string{"ARGS: one two", "on stderr", "STDOUT: file"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunRequiresAnOutputFile(t *testing.T) {
	var r procrun.Runner
	if err := r.Run(t.Context(), &procrun.Call{Name: "true"}); err == nil {
		t.Fatal("a Call without Output must be refused")
	}
}

func TestRunAppends(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)
	out := filepath.Join(dir, "run.log")

	var r procrun.Runner
	for i := range 2 {
		call := &procrun.Call{Name: bin, Args: []string{"round"}, Output: out, Append: i > 0}
		if err := r.Run(t.Context(), call); err != nil {
			t.Fatalf("round %d: %v", i, err)
		}
	}

	if n := strings.Count(read(t, out), "ARGS: round"); n != 2 {
		t.Fatalf("appended output holds %d rounds, want 2", n)
	}

	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Args: []string{"round"}, Output: out}); err != nil {
		t.Fatal(err)
	}

	if n := strings.Count(read(t, out), "ARGS: round"); n != 1 {
		t.Fatalf("without Append the file holds %d rounds, want 1", n)
	}
}

func TestScrubDropsInheritedFamilies(t *testing.T) {
	t.Setenv("VENDOR_TOKEN", "inherited")
	t.Setenv("VENDOR_PROJECT", "inherited")
	t.Setenv("OTHER_SETTING", "kept")

	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)
	out := filepath.Join(dir, "run.log")

	r := procrun.Runner{
		Scrub: []string{vendorPrefix},
		Env:   map[string]string{envLocale: "C"},
	}
	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Output: out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := read(t, out)
	for _, gone := range []string{"ENV VENDOR_TOKEN=", "ENV VENDOR_PROJECT="} {
		if strings.Contains(got, gone) {
			t.Fatalf("scrubbed variable leaked: %q\n%s", gone, got)
		}
	}

	for _, want := range []string{"ENV OTHER_SETTING=kept", "ENV LC_ALL=C"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in the environment:\n%s", want, got)
		}
	}
}

func TestCallEnvReachesOnlyThatCall(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)

	r := procrun.Runner{Scrub: []string{vendorPrefix}}

	withVar := filepath.Join(dir, "with.log")
	call := &procrun.Call{Name: bin, Env: map[string]string{envIdentity: identityReader}, Output: withVar}

	if err := r.Run(t.Context(), call); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(read(t, withVar), "ENV VENDOR_IDENTITY=reader") {
		t.Fatal("a per-call variable must reach that call")
	}

	without := filepath.Join(dir, "without.log")
	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Output: without}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(read(t, without), "ENV VENDOR_IDENTITY=") {
		t.Fatal("a per-call variable must not reach the next call")
	}
}

func TestFixedEnvOverridesInherited(t *testing.T) {
	t.Setenv(envLocale, "en_US.UTF-8")

	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)
	out := filepath.Join(dir, "run.log")

	r := procrun.Runner{Env: map[string]string{envLocale: "C"}}
	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Output: out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := read(t, out)
	if strings.Contains(got, "ENV LC_ALL=en_US.UTF-8") {
		t.Fatalf("the inherited value must be replaced, not kept:\n%s", got)
	}

	if !strings.Contains(got, "ENV LC_ALL=C") {
		t.Fatalf("the fixed value must be set:\n%s", got)
	}
}

func TestExitCodeIsCarried(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "run.log")

	for _, code := range []int{1, 2, 7} {
		bin := stub(t, dir, "tool"+strconv.Itoa(code), code)

		err := (&procrun.Runner{}).Run(t.Context(), &procrun.Call{Name: bin, Output: out})
		if err == nil {
			t.Fatalf("exit code %d must produce an error", code)
		}

		if got := procrun.ExitCodeOf(err); got != code {
			t.Fatalf("ExitCodeOf = %d, want %d", got, code)
		}

		var ee *procrun.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("error is not an *ExitError: %T", err)
		}

		if ee.Output != out || !strings.Contains(ee.Tail, "on stderr") {
			t.Fatalf("the error must carry the output tail: %+v", ee)
		}

		if !strings.Contains(err.Error(), "output: "+out) {
			t.Fatalf("the message must name the output file: %q", err.Error())
		}
	}
}

func TestExitCodeOfOtherErrors(t *testing.T) {
	if got := procrun.ExitCodeOf(nil); got != -1 {
		t.Fatalf("ExitCodeOf(nil) = %d, want -1", got)
	}

	if got := procrun.ExitCodeOf(errors.New("boom")); got != -1 {
		t.Fatalf("ExitCodeOf(other) = %d, want -1", got)
	}
}

func TestTailLines(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 3)
	out := filepath.Join(dir, "run.log")

	err := (&procrun.Runner{TailLines: 1}).Run(t.Context(), &procrun.Call{Name: bin, Output: out})

	var ee *procrun.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("want an *ExitError, got %T", err)
	}

	if strings.Contains(ee.Tail, "\n") {
		t.Fatalf("TailLines 1 must give one line, got %q", ee.Tail)
	}

	err = (&procrun.Runner{TailLines: -1}).Run(t.Context(), &procrun.Call{Name: bin, Output: out})
	if !errors.As(err, &ee) {
		t.Fatalf("want an *ExitError, got %T", err)
	}

	if ee.Tail != "" {
		t.Fatalf("a negative TailLines must give no tail, got %q", ee.Tail)
	}
}

func TestLookPathUsesTheFixedPath(t *testing.T) {
	dir := t.TempDir()
	stub(t, dir, "tool", 0)

	r := procrun.Runner{Path: dir}

	got, err := r.LookPath("tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filepath.Join(dir, "tool") {
		t.Fatalf("LookPath = %q", got)
	}

	// A name that is not on the fixed PATH is not found, even when the
	// parent's PATH would have resolved it.
	t.Setenv("PATH", dir)

	if _, err := (&procrun.Runner{Path: t.TempDir()}).LookPath("tool"); !errors.Is(err, procrun.ErrNotOnPath) {
		t.Fatalf("want ErrNotOnPath, got %v", err)
	}
}

func TestLookPathPassesThroughExplicitPaths(t *testing.T) {
	r := procrun.Runner{Path: t.TempDir()}

	got, err := r.LookPath(shellPath)
	if err != nil || got != shellPath {
		t.Fatalf("LookPath(/bin/sh) = %q, %v", got, err)
	}
}

func TestLookPathFallsBackToTheParentPath(t *testing.T) {
	dir := t.TempDir()
	stub(t, dir, "tool", 0)
	t.Setenv("PATH", dir)

	got, err := (&procrun.Runner{}).LookPath("tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filepath.Join(dir, "tool") {
		t.Fatalf("LookPath = %q", got)
	}

	if _, err := (&procrun.Runner{}).LookPath("no-such-command-here"); err == nil {
		t.Fatal("an unknown command must not resolve")
	}
}

func TestPathIsHandedToTheChild(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)
	out := filepath.Join(dir, "run.log")

	r := procrun.Runner{Path: fixedPath}
	if err := r.Run(t.Context(), &procrun.Call{Name: bin, Output: out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(read(t, out), "ENV PATH=/usr/bin:/bin") {
		t.Fatal("the child must receive the fixed PATH")
	}
}

func TestCaptureReturnsStdoutOnly(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 0)

	out, err := (&procrun.Runner{}).Capture(t.Context(), &procrun.Call{Name: bin, Args: []string{"version"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "ARGS: version") {
		t.Fatalf("stdout = %q", out)
	}

	if strings.Contains(out, "on stderr") {
		t.Fatal("Capture returns standard output only")
	}
}

func TestCaptureCarriesTheExitCode(t *testing.T) {
	dir := t.TempDir()
	bin := stub(t, dir, "tool", 4)

	_, err := (&procrun.Runner{}).Capture(t.Context(), &procrun.Call{Name: bin})
	if got := procrun.ExitCodeOf(err); got != 4 {
		t.Fatalf("ExitCodeOf = %d, want 4 (%v)", got, err)
	}
}

func TestMissingBinaryIsNotAnExitError(t *testing.T) {
	dir := t.TempDir()

	err := (&procrun.Runner{}).Run(t.Context(), &procrun.Call{
		Name: filepath.Join(dir, "absent"), Output: filepath.Join(dir, "run.log"),
	})
	if err == nil {
		t.Fatal("a missing binary must produce an error")
	}

	if procrun.ExitCodeOf(err) != -1 {
		t.Fatalf("a missing binary carries no exit code: %v", err)
	}
}

func TestUnwritableOutputIsReported(t *testing.T) {
	err := (&procrun.Runner{}).Run(t.Context(), &procrun.Call{
		Name: shellPath, Args: []string{"-c", "true"}, Output: filepath.Join(t.TempDir(), "no-dir", "run.log"),
	})
	if err == nil {
		t.Fatal("an unwritable output file must produce an error")
	}
}

func TestEnvironIsInspectable(t *testing.T) {
	t.Setenv("VENDOR_TOKEN", "inherited")

	r := procrun.Runner{Path: "/usr/bin", Scrub: []string{vendorPrefix}, Env: map[string]string{envLocale: "C"}}

	env := r.Environ(map[string]string{envIdentity: identityReader})

	var sawPath, sawLocale, sawIdentity bool

	for _, kv := range env {
		switch kv {
		case "PATH=/usr/bin":
			sawPath = true
		case "LC_ALL=C":
			sawLocale = true
		case "VENDOR_IDENTITY=reader":
			sawIdentity = true
		}

		if kv == "VENDOR_TOKEN=inherited" {
			t.Fatal("a scrubbed variable must not survive")
		}
	}

	if !sawPath || !sawLocale || !sawIdentity {
		t.Fatalf("Environ = %v", env)
	}

	// A per-call variable must win over the same name in Env.
	r.Env = map[string]string{envIdentity: "writer"}
	for _, kv := range r.Environ(map[string]string{envIdentity: identityReader}) {
		if kv == "VENDOR_IDENTITY=writer" {
			t.Fatal("the per-call value must win")
		}
	}
}

func TestContextCancellationStopsTheCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := (&procrun.Runner{}).Run(ctx, &procrun.Call{
		Name: shellPath, Args: []string{"-c", "sleep 5"}, Output: filepath.Join(t.TempDir(), "run.log"),
	})
	if err == nil {
		t.Fatal("a cancelled context must stop the command")
	}
}
