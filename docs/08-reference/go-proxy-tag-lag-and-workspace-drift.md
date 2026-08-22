---
uuid: 8ca68e51-588f-4a45-9369-a9f3173c95f7
title: "Go Proxy Tag Lag and Workspace Drift"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T09:00:00Z"
updated_at: "2026-08-22T09:00:00Z"
summary: >
  The Go module proxy's version-list endpoint can lag behind a freshly
  pushed tag, and an ambient local go.work file masks version-resolution
  bugs that only appear in CI; verify a specific tagged version directly
  and pin GOWORK=off in build scripts that must match CI.
audience: [public]
tags: [ "go", "modules", "goproxy", "go-work", "ci", "lessons-learned" ]
relations: []
---

# Go Proxy Tag Lag and Workspace Drift

Two independent Go tooling behaviors compound to make "it works on my
machine, right after I tagged it" an unreliable signal: proxy propagation
delay, and an uncommitted local workspace file overriding module
resolution.

## Symptom

Immediately after tagging and pushing a new module version, `go list -m
-versions <module>` does not list the new tag, even though the tag exists
on the remote and its per-version metadata already resolves. Separately, a
build that passes locally (against local module source via a `go.work`
file) fails or resolves a different version once run in CI or by another
contributor who has no such workspace file.

## Cause

- **Proxy lag**: the Go module proxy's `@v/list` endpoint (what `go list
  -m -versions` reads) is populated asynchronously and can lag behind a
  just-pushed tag, even though the tag's own `@v/<version>.info` endpoint
  already resolves correctly. Checking only the list endpoint produces a
  false negative for "is this version published yet."
- **Workspace drift**: a `go.work` file overrides `go.mod` version pins
  for every module it lists, redirecting builds to local source. If it
  exists only on one machine (uncommitted, as intended for local
  cross-module development) and a build script doesn't explicitly disable
  it, that script silently gets different resolution behavior locally
  than CI does — masking a real version-pinning bug until it surfaces
  downstream.

## Rule

- Verify a freshly pushed tag by resolving the **specific version**
  directly — `GOPROXY=<proxy> go list -m <module>@<version>` or a direct
  fetch of the version's `.info` endpoint — never by checking only the
  `-versions` (list) output, which can lag.
- Any build or test script that needs to match CI's resolution behavior
  should set `GOWORK=off` explicitly, rather than relying on the absence
  of a `go.work` file. This makes the script's behavior independent of
  whichever machine happens to run it.
- If a local `go.work` is needed for cross-module development, generate
  it from a small script (or keep it gitignored and documented) rather
  than hand-maintaining it, and regenerate it whenever a module is added
  or removed — a stale workspace silently drops a module back to its
  pinned (possibly older) published version.

## How to Test for It

- After tagging a new version, run both `go list -m -versions <module>`
  and `go list -m <module>@<version>` in the same script; a mismatch
  (the specific version resolves, but doesn't appear in the list) confirms
  proxy lag rather than a real publication failure.
- Run the same build twice in the same environment — once with the
  ambient `go.work` in place, once with `GOWORK=off` — and diff the
  resolved module versions (`go list -m all` in both modes). Any
  difference means a script that omits `GOWORK=off` will not match CI.
- In CI, add a step that asserts no `go.work` file is present (or that
  `GOWORK` is explicitly `off`) before running the module test matrix, so
  a workspace file accidentally committed doesn't silently change what CI
  builds.
