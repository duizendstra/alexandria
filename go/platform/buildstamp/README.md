# go/platform/buildstamp

`go/platform/buildstamp` records which build produced a binary, and verifies that a binary is the one you meant to run.

When a tool changes production state, "which build was that?" decides whether an incident can be reconstructed. A stamp answers it: the commit, whether the tree was clean, when it was built, and the provenance of dependencies the module graph does not pin.

## Features

- **Stamp on build, verify at run**: `Get` reads the running binary's identity from `-ldflags` values, falling back to Go's embedded VCS settings. `ParseStamp` reads a stamp back out of a version line, so a wrapper or supervisor can verify a binary it did not build.
- **Refusals that explain themselves**: `Matches` returns a descriptive error, not a bool — the caller usually needs to tell an operator what to rebuild and why.
- **Strict by default**: a full 40-character SHA is required (abbreviated SHAs are ambiguous), `unknown` is never accepted, a dirty tree is refused, and recorded dependency stamps must be clean — a binary built against a dirty local dependency is not pinned.
- **`RequireDeps` for provenance you insist on**: `Matches` can only judge the stamps that are present; `RequireDeps` catches a build that forgot to record one.
- **Stable output**: dependency stamps are rendered in sorted order, so the line can be compared byte for byte between runs.
- **Zero dependencies** outside the standard library.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/buildstamp
```

## Quick Start

### Stamping a binary

```go
var deps = map[string]string{"lib": libRevision} // provenance you track yourself

func main() {
	stamp := buildstamp.Get("tool", deps)
	fmt.Println(stamp.String()) // tool 1.4.0 commit=<sha> dirty=false built=… lib=abc1234
}
```

Build it with the values filled in:

```bash
go build -ldflags "
  -X github.com/duizendstra/alexandria/go/platform/buildstamp.Commit=$(git rev-parse HEAD)
  -X github.com/duizendstra/alexandria/go/platform/buildstamp.Modified=false
  -X github.com/duizendstra/alexandria/go/platform/buildstamp.BuiltAt=$(date -u +%FT%TZ)"
```

### Verifying before you let it run

```go
stamp, err := buildstamp.ParseStamp(versionOutput)
if err != nil {
	return fmt.Errorf("unreadable build stamp: %w", err)
}
if err := stamp.Matches(releaseSHA); err != nil {
	return err // e.g. "built from a dirty working tree — rebuild from a clean checkout"
}
if err := stamp.RequireDeps("lib"); err != nil {
	return err
}
```

## Design Notes

The strictness is deliberate. Each rule closes a way a build can look trustworthy while not being reproducible:

| Rule | Why |
|---|---|
| Full 40-character SHA | An abbreviated SHA is ambiguous, and ambiguity is what a stamp exists to remove. |
| `unknown` is never accepted | It is what you get when the build lost its VCS context — exactly when guessing is worst. |
| Dirty tree refused | Uncommitted changes cannot be reconstructed later. |
| Dependency stamps must be clean | A pinned binary built against a dirty local dependency is not pinned. |

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Operator tools that change production state**: binaries where "which build
  was that?" has to be answerable after the fact.
- **Two-step ceremonies**: commands whose apply step must refuse a plan
  produced by a different build.

### Load-Bearing Promises
1. **Round-Trip Is Lossless**: a stamp rendered with `String` parses back to an
   equal stamp.
2. **Rendering Is Deterministic**: the same stamp renders identically
   regardless of dependency ordering, so stamps can be compared as text.
3. **Unknown Keys Are Kept**: a stamp written by a newer version parses rather
   than failing, with unrecognised keys retained as dependencies. Malformed
   lines are still rejected.
4. **`Matches` Only Accepts A Clean Exact Build**: a dirty tree, a different
   commit, or a missing stamp is a refusal, and the refusal says which.
5. **Absence Degrades To `unknown`**: a binary built without stamping reports
   `unknown` instead of failing at startup.
6. **`Get` Copies**: the returned dependency set is a copy, so a caller cannot
   mutate the recorded stamp.
