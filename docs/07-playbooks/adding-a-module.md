---
title: Adding a New Module
domain: playbooks
type: guide
diataxis_quadrant: how-to
status: draft
maturity: draft
audience: [public]
owner: "@duizendstra"
summary: How to scaffold a new Go module under go/, author it to the zero-rot package standard, wire it into CI (module index, Dependabot, coverage baseline), verify it locally, and release it with a path-prefixed tag.
uuid: 8cdf6842-b989-4ec1-8bf5-5666d8ab962e
created_at: "2026-08-22T16:02:30Z"
updated_at: "2026-08-22T16:02:30Z"
tags: [ "playbook", "go", "module", "scaffold", "ci", "release" ]
relations:
  - target_uuid: "4a2d8e41-6fb7-4b42-a89e-26f6323c9de2"
    rel_type: "relates_to"
  - target_uuid: "b5ca5a05-41c7-49cb-8cb6-06ea4258c90b"
    rel_type: "relates_to"
  - target_uuid: "cd2acade-1fc9-4423-896b-07681165d1e1"
    rel_type: "relates_to"
  - target_uuid: "599a1bf7-9a16-4369-8574-4126261e7578"
    rel_type: "relates_to"
---

# Adding a New Module

This guide walks a new Go module from an empty directory to a released,
CI-wired, published package. The canonical checklist lives in
[CONTRIBUTING.md](../../CONTRIBUTING.md); this playbook is the ordered how-to
that surrounds it — including the CI wiring steps the checklist assumes but
does not spell out.

## When to use this

Reach for this when you are creating a brand-new module under `go/` — a fresh
`go.mod` with its own version lifecycle. If you are instead moving an existing
internal package into Alexandria, start from the migration guidance in
[module-adoption.md](module-adoption.md); the scaffolding and wiring steps
below still apply once the code lands.

## Before you start — does this belong in Alexandria?

Alexandria is **public** ([ADR-0001](../04-decisions/adr-0001-monorepo-strategy.md)).
Only generic, portable code belongs here; anything carrying a client, company,
project, or team name does not — that code stays in its private consumer. Settle
this first: the mechanical steps below are wasted effort on code that can never
be published.

If a shared module would block a consumer, the answer is not a workaround —
build the primitive in the consuming service's own `lib/` now and ascend it
later.

## Mental model

Each directory under `go/` is an **independent module**: its own `go.mod`, its
own path-prefixed version tags (`go/<module>/vX.Y.Z`), its own coverage
baseline. There are no committed `replace` directives — modules depend on each
other through published tags, and cross-module work happens in an uncommitted
`go.work`. A new module is therefore four things: source, a CI wiring triplet
(index + Dependabot + coverage baseline), local green, and a tag.

## Procedure

### 1. Scaffold the module

```bash
mkdir go/my-package
cd go/my-package
go mod init github.com/duizendstra/alexandria/go/my-package
```

Nest deeper when the domain calls for it (`go/platform/my-package`,
`go/google/workspace/my-package`); the import path and every later step use the
full path.

### 2. Author to the zero-rot package standard

Follow the step-by-step blueprint in
[writing-enterprise-go-packages.md](../03-architecture/writing-enterprise-go-packages.md).
At minimum every module ships:

- `doc.go` — package-level documentation.
- `example_test.go` — at least one `Example` function that compiles against the
  real API and whose output matches the actual behaviour.
- `README.md` — install + usage for the module.

### 3. Wire it into CI

Three edits, all easy to forget and all enforced downstream:

1. **Module index** — add the module to the table in the root
   [README.md](../../README.md).
2. **Dependabot** — add a `gomod` entry in
   [`.github/dependabot.yml`](../../.github/dependabot.yml) with
   `directory: "/go/my-package/"`.
3. **Coverage baseline** — add a `"go/my-package": <n>` entry to
   [`.github/coverage-baselines.json`](../../.github/coverage-baselines.json).
   Start at `0` (or ~2 points under measured coverage) and ratchet up; the CI
   `go` job fails if measured coverage drops below the recorded baseline.
   Generated-code modules are exempt (see the file's `_comment`).

### 4. Verify locally

The module discovery used by `just` mirrors the CI matrix (`find go -name
go.mod`), so a green `just check` locally means a green matrix:

```bash
just check      # vet-all + lint-all + test-all + fuzz-all + check-versions
```

Without `just`, isolate the module the same way CI does:

```bash
cd go/my-package
GOWORK=off go test -race -count=1 ./...
GOWORK=off golangci-lint run ./...
```

Lint runs the **library profile** for platform modules; consumer-shaped modules
have their own profile. For recurring lint fixes see
[golangci-resolutions.md](golangci-resolutions.md).

### 5. Document it in the vault (if it warrants a doc)

A module's own `README.md`/`doc.go` are usually enough. Add a `docs/` concept
only when there is architecture or a decision worth capturing — and if you do,
it must satisfy the OKF frontmatter schema enforced by
[`scripts/okf-lint.py`](../../scripts/okf-lint.py) (see the
[OKF profile](../08-reference/okf-profile.md)).

### 6. Release

Tag in dependency order with a path-prefixed tag, then verify the proxy picked
it up:

```bash
git tag go/my-package/v0.1.0
git push origin go/my-package/v0.1.0
```

For a change that spans this module and its consumers, follow the staged pins
and dependency-ordered tagging in
[cross-module-release.md](cross-module-release.md).

## Publication gate

Before a module is considered public-ready, it must clear the **Publication
Checklist** in [CONTRIBUTING.md](../../CONTRIBUTING.md) — zero client/company
references, ≥80% coverage, `doc.go` + `example_test.go`, Apache-2.0 header,
`golangci-lint` clean, README examples that compile. That checklist is the
single source of truth; this playbook deliberately does not duplicate it.

## Gotchas

- **No committed `replace` directives.** They mask tagged versions and break
  reproducible builds. Use an uncommitted `go.work` for local cross-module work.
- **Keep the index honest.** A module missing from the root README table, from
  Dependabot, or from the coverage baseline is invisible to the people and jobs
  that rely on those files.
- **Proxy tag lag.** A freshly pushed tag can 404 on the module proxy for a
  short window; see
  [go-proxy-tag-lag-and-workspace-drift.md](../08-reference/go-proxy-tag-lag-and-workspace-drift.md).
- **Conventional Commits.** Scope commits to the module
  (`feat(my-package): ...`); the `commit-msg` hook rejects non-conforming
  messages.

## Related

- [writing-enterprise-go-packages.md](../03-architecture/writing-enterprise-go-packages.md) — the package-quality standard behind step 2.
- [cross-module-release.md](cross-module-release.md) — releasing a change that spans a module and its consumers.
- [module-adoption.md](module-adoption.md) — the downstream side: adopting a published module.
- [ADR-0001](../04-decisions/adr-0001-monorepo-strategy.md) — why the multi-module monorepo, and the independent-versioning policy.
