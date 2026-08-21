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

## Consumers

- Orchestrators that drive other binaries as a permission boundary: the child
  process is the boundary, so an identity can only exist inside the process
  that is allowed to have it.
