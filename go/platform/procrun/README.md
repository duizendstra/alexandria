# go/platform/procrun

Run external commands under an environment you fully control.

A tool that shells out to other binaries usually cares about three things that
`os/exec` does not do by itself:

- **No inherited variable may redirect the child.** Credential paths, project
  selection and locale are exactly the kind of setting that a developer's shell
  carries around and that silently changes what a command does.
- **The binary that runs is the one on a known PATH.** `exec.Command` resolves
  a bare name against the *parent's* `PATH`, not against whatever you put in
  `cmd.Env` — so setting a fixed `PATH` in the environment is not enough.
- **Output goes to a file, not a pipe.** A reader that stops early closes the
  pipe, the child dies on `SIGPIPE`, and a complete result becomes a truncated
  one with nothing to show for it.

```go
r := procrun.Runner{
    Path:  "/usr/bin:/bin",
    Env:   map[string]string{"LC_ALL": "C"},
    Scrub: []string{"VENDOR_"},          // drop every inherited VENDOR_* variable
}

err := r.Run(ctx, &procrun.Call{
    Name:   "some-tool",
    Args:   []string{"report", "--out", path},
    Env:    map[string]string{"VENDOR_IDENTITY": "reader"}, // this call only
    Output: filepath.Join(logDir, "some-tool.log"),
})

if code := procrun.ExitCodeOf(err); code > 0 {
    // map the tool's exit code onto your own classes
}
```

## API

- `Runner.Run(ctx, *Call) error` — runs the command with stdout and stderr on
  `Call.Output`.
- `Runner.Capture(ctx, *Call) (string, error)` — returns standard output; for
  short, fully-read output such as a version banner.
- `Runner.Environ(callEnv) []string` — the exact environment a call would
  receive, so tests can assert on it.
- `Runner.LookPath(name) (string, error)` — resolution against the fixed
  `Path`; returns `ErrNotOnPath` when the name is not there.
- `ExitCodeOf(err) int` — the child's exit code, or `-1` when the error carries
  none (a command that never started).

A non-zero exit gives an `*ExitError` with the code, the output path and the
last `TailLines` lines of that file (20 by default) — enough to put in a
message without reading the log again.

The zero `Runner` inherits the parent environment unchanged and resolves
commands with the parent's `PATH`; set the fields to tighten that.

Zero dependencies beyond the standard library.

## Install

```bash
go get github.com/duizendstra/alexandria/go/platform/procrun
```

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Orchestrators that drive other binaries as a permission boundary**: the
  child process is the boundary, so an identity can only exist inside the
  process that is allowed to have it.
- **Reproducible build and migration steps**: callers that need a child to
  behave the same on a developer laptop and on a runner.

### Load-Bearing Promises
1. **Nothing Is Inherited By Accident**: inherited environment families are
   dropped rather than passed through; a variable reaches the child only
   because it was allow-listed or fixed.
2. **A Fixed Value Replaces, Not Merges**: an entry in `Runner.Env` overrides
   the inherited value of the same name — setting `LC_ALL=C` displaces an
   inherited `en_US.UTF-8` rather than losing to it.
3. **Per-Call Env Stays In That Call**: environment set on a `Call` reaches
   only that call and does not leak into later ones through the `Runner`.
4. **Output Goes To A File, Never A Pipe**: both streams are written to the
   named output file, and a `Call` without one is refused rather than run.
   An unwritable output path is reported as an error.
5. **Exit Codes Are Carried, Absence Is Not An Exit**: a child's non-zero exit
   arrives as an `ExitError` carrying the code, while a missing binary is a
   different error — so "failed" and "never ran" are distinguishable.
6. **`PATH` Is The Fixed One**: lookup and the child both see the configured
   `PATH`, not the parent's, and explicit paths pass through untouched.
7. **Windows Resolves Through `PATHEXT`**: extensions are appended in
   `PATHEXT` order, earlier entries win, a non-executable extension is
   rejected, and an unset `PATHEXT` falls back to a default.
