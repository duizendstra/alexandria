---
uuid: 2b1be1ab-1cc5-4402-ac14-2797286c7b88
title: "golangci-lint Resolutions Cheat-Sheet"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "draft"
owner: "@duizendstra"
created_at: "2026-07-21T00:00:00Z"
updated_at: "2026-07-21T00:00:00Z"
summary: >
  Recurring fixes for getting a Go change to 0 issues under the Alexandria
  golangci profiles: the library vs consumer posture, the gocritic/nonamedreturns
  struct pattern, err113 sentinels, the pulumi forcetypeassert rewrite, and the
  goconst-counts-tests trap.
audience: [public]
tags: [ "golangci-lint", "linting", "go", "pulumi", "playbook" ]
relations: []
---

# golangci-lint resolutions

A cheat-sheet for getting a Go change to **0 issues** under the Alexandria
golangci profiles (`max-issues-per-linter: 0`, `nolintlint` requires
specific + explained suppressions — nothing is "one warning, ship it").
Covers the recurring fixes that aren't obvious from the linter message.

## Which profile am I under?

There are two golden profiles in `blueprints/golangci/`, one quality bar,
two dependency postures:

| Profile | Governs | funlen | cyclop | nestif | maintidx `under` | lll | depguard |
|---|---|---|---|---|---|---|---|
| **library** (`library.golangci.yml`) | reusable library modules | 120 / 60 | 15 | 5 | 15 | 160 | stdlib + curated externals (incl. pulumi) |
| **consumer** (`consumer.golangci.yml`) | apps, services, Pulumi stacks, CLIs | 80 / 40 | 10 | 4 | 20 | 120 | stdlib + Alexandria library modules only |

**Alexandria's own `.golangci.yml` (repo root) is a library-profile
instance**, and it is the *only* lint config in the repo: it governs every
module under `go/` — **including `go/iac/pulumi/gcpinfra`**. There is no
separate stricter profile for the pulumi/gcpinfra code; the pulumi
resolutions below run under this same root library config.

The consumer profile matters when you copy a blueprint into a *downstream*
composition root. Two traps when moving between profiles:

- **Complexity ceilings drop** (funlen 120→80, cyclop 15→10). Library code
  that passes can trip the consumer limits — extract sub-functions.
- **depguard tightens.** The consumer allowlist is stdlib +
  `github.com/duizendstra/alexandria/go` only. Any other external import
  fails closed; that's the signal the code belongs in a library module, not
  in the composition root.

---

## Library-flavored resolutions

These apply to any module under the ~60-linter library bar (all of
Alexandria's `go/`, and downstream library repos).

### `gocritic: unnamedResult` vs `nonamedreturns` — the house pattern

The trap: a function returning **≥3 values**, or **2 of the same type**
(`(int, int)`, `(string, string, error)`), trips `unnamedResult` (it wants
names for clarity). But **naming** the returns trips `nonamedreturns`. You
can't satisfy both by fiddling with the signature.

Escape by returning a small named **struct**:

```go
type repoRef struct {
    repoID     string
    taskTypeID string
}

func resolve(...) (repoRef, error) {
    ...
    return repoRef{repoID: id, taskTypeID: tt}, nil
}
```

One documented return value, self-describing fields, both linters quiet.
This is the standard fix — reach for it before any `//nolint`.

### `gocritic: hugeParam` (and `recvcheck`)

Fires when a struct larger than ~80 bytes is passed **by value**. Fixes:

- Pass/receive by pointer: `func (s *Server) ...`, `func do(cfg *Config)`.
- Iterate slices **by index**, not value-copy range:

  ```go
  for i := range xs {
      x := &xs[i]
      // use x
  }
  ```

- Keep receivers **consistent** across a type — all pointer or all value.
  Mixing them trips `recvcheck`.

### `err113` — no bare dynamic errors

Every error must be a wrapped sentinel or a static message. A bare
`fmt.Errorf("bad arg %s", x)` (dynamic, no `%w`) fails.

- Declare sentinels in an `errors.go`: `var errUsage = errors.New("...")`,
  `errGraphQL`, etc.
- Wrap with `%w`: `fmt.Errorf("parsing %s: %w", name, errUsage)`.

### `gosec: G602` — slice index after a manual bounds check

`gosec` can't prove an index is in range even right after you checked it.
When the check is directly above, suppress with a specific, reasoned nolint:

```go
if i >= len(args) {
    return errUsage
}
val := args[i] //nolint:gosec // G602: i < len(args), guaranteed by the check above
```

### `nilnil` — legitimate `(nil, nil)`

Returning `(nil, nil)` for "no result, no error" (e.g. a cache miss) is
valid but flagged. Suppress with a reason:

```go
return nil, nil //nolint:nilnil // cache miss is not an error
```

### `cyclop` — complexity ceiling (library max 15)

Over the limit → extract a sub-function. Don't raise the ceiling. (Consumer
profile is 10 — the same code may need splitting when it moves into an app.)

### Modernizers: `intrange` / `modernize` / `mirror` / `perfsprint`

Take the modern form:

- `for i := range n` instead of `for i := 0; i < n; i++`.
- `strings.IndexByte(s, 'x')` over `strings.Index(s, "x")` (`mirror`).
- `strings.Cut` over `Split`/`SplitN` for single-separator parsing.

**Exception:** keep a manual `i := 0; for i < len(args) { ... }` while-loop
when the body must **advance `i` past a consumed value** (e.g. a flag and
its argument). `intrange` doesn't apply to a loop whose index you mutate.

### Intentional shell-out: `git()`

A short-lived local `git()` helper that shells out carries a documented
combined suppression:

```go
cmd := exec.Command("git", args...) //nolint:gosec,noctx // trusted local git, no ctx needed
```

---

## Pulumi / gcpinfra resolutions (under the root library profile)

Building-block code in `go/iac/pulumi/gcpinfra/*` is linted by the root
library config — same profile, but these patterns are specific to Pulumi.

### `forcetypeassert` on the `ApplyT` string idiom — don't nolint, rewrite

Building a string output through `ApplyT` ends in an **unchecked type
assertion**, which `forcetypeassert` (enabled in all profiles) flags:

```go
// FLAGGED: trailing .(pulumi.StringOutput) is unchecked
url := out.ApplyT(func(s string) string {
    return "https://" + s + "/api"
}).(pulumi.StringOutput)
```

Fix by replacing the whole idiom with `pulumi.Sprintf` — it takes inputs,
returns `pulumi.StringOutput` directly, and needs no assertion:

```go
url := pulumi.Sprintf("https://%s/api", out)
```

No `//nolint` — the idiom itself is the smell.

### `mnd` — magic numbers (e.g. ports)

A bare literal like `443` fails `mnd`. Hoist to a named const:

```go
const httpsPort = 443
```

### `gochecknoglobals` vs a slice "default"

`mnd`/readability pushes you to name a default set, but a slice can't be
`const`, and a package-level `var defaults = []string{...}` trips
`gochecknoglobals`. Resolution: keep the **scalar** parts as `const` and
build the slice **at its single use inside the function** — no package-level
`var`:

```go
const defaultRole = "roles/viewer"

func apply(...) {
    roles := []string{defaultRole, "roles/logging.viewer"}
    ...
}
```

### goconst and `_test.go` — THE trap, and it is NOT profile-symmetric

`goconst` is enabled in every profile, and **no Alexandria profile sets
`goconst.ignore-tests`** — not the library blueprint, not the consumer
blueprint, not the repo-root config. Consequence: **Alexandria counts
repeated literals inside `_test.go` too.** A value used ≥3× across, say,
`config_test.go` + `example_test.go` must become a shared `const` in the
`_test` package:

```go
// in a shared _test file
const testProjectID = "example-project"
```

Do **not** assume the exemption exists. A *downstream* consumer repo may
add `goconst: { ignore-tests: true }` to its own copy of a blueprint (some
do), which makes test-data repetition free there — but that is a per-repo
customization, not inherited from these profiles. Check the config in the
repo you're actually in. In Alexandria, test literals count; genuinely
shared non-test strings always become consts regardless.

---

## Verify green

Run from the module you touched (each `go/` dir is its own module):

```bash
GOWORK=off golangci-lint run ./...     # must print "0 issues"
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test ./...
```

`GOWORK=off` matters: a local `go.work` can mask a single-module failure
that CI (which runs each module in isolation) will catch.

**Every module's CI** (`.github/workflows/ci.yml`, matrix over
`find go -name go.mod`) additionally runs, per module:

```bash
GOWORK=off go test -race -count=1 -coverprofile=coverage.out ./...
```

and a **coverage ratchet**: measured coverage must stay at or above the
module's baseline in `.github/coverage-baselines.json` (e.g.
`go/iac/pulumi/gcpinfra: 60`; `go/contracts` is exempt as generated code).
Reproduce it locally before pushing:

```bash
just check        # vet + lint + test across every module (needs the Nix dev shell)
just cover-all    # per-module coverage summary vs baselines
```

Lower a baseline only as a recorded, deliberate decision; the long-term
target is the 80% publication bar (`CONTRIBUTING.md`).

**Windows-targeting CLI code** (library-profile CLIs that must build on
Windows too) — add a cross-compile smoke check:

```bash
GOOS=windows GOARCH=amd64 go build ./...
```
