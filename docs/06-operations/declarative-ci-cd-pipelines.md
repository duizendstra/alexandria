---
uuid: ef14ea7c-5f4b-40dd-a407-276836e3fd11
title: "Declarative CI/CD Pipelines & Release Automation"
domain: "operations"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-04T09:00:00Z"
updated_at: "2026-07-19T12:00:00Z"
summary: >
  Defines the continuous integration rules, release workflows, and semantic tagging
  standards for multi-module monorepos, distinguishing the pipeline that runs today
  from planned automation.
audience: [public]
tags: [ "operations", "ci-cd", "tagging" ]
relations:
  - target_uuid: "b4bc306c-9ba5-4eb8-b99b-efb829623dc1"
    rel_type: "depends_on"
---
# Declarative CI/CD Pipelines & Release Automation

## Operational Objective

To establish a continuous integration and automated release delivery model that validates code quality, ensures contract compatibility, and handles multi-module semantic tagging with minimal manual intervention.

This document separates the **current pipeline** (what `.github/workflows/ci.yml` actually runs) from **planned automation** (checks we intend to add but which are not yet enforced). Treat only the former as a gate you can rely on.

---

## The Multi-Module Monorepo Release Problem

In a monorepo housing several independent Go modules (such as `/go/retry` and `/go/slog-gcp`), standard repository-wide versioning is an anti-pattern. 
Consumers importing `go/retry` should not have their dependency bumped because of changes in `go/slog-gcp`.

Therefore, we enforce **path-prefixed multi-module semantic versioning**. Each subdirectory acts as an independent release boundary.

---

## Current Pipeline

`.github/workflows/ci.yml` executes the following jobs on every Pull Request targeting the `main` branch (and on pushes to `main`):

```
[ Pull Request ]
       |
       v
+------------------+     +----------------------------------+
| Detect Modules   | --> | Per-module: vet / test / lint    |
| (find go.mod)    |     | (go vet, go test -race, golangci)|
+------------------+     +----------------------------------+

+------------------+     +------------------+     +------------------+
| Module Hygiene   |     | Contracts (buf)  |     | Docs Link Check  |
| (mod-hygiene)    |     | lint/break/drift |     | (relative links) |
+------------------+     +------------------+     +------------------+
```

1.  **Module Discovery** — Dynamically finds every `go.mod` under `go/` so new modules are tested without pipeline edits.
2.  **Per-Module Vet, Test & Lint** — For each module: `go vet ./...`, `go test -race -count=1 -coverprofile=coverage.out ./...`, and `golangci-lint`. Any failure blocks the merge. Coverage is *collected* but no minimum percentage is enforced.
3.  **Module Hygiene** (`mod-hygiene`, introduced by PR [#37](https://github.com/duizendstra/alexandria/pull/37)) — Rejects committed `replace` directives, unresolvable `v0.0.0` pins, and modules missing Dependabot coverage.
4.  **Contracts** (`contracts`, introduced by PR [#38](https://github.com/duizendstra/alexandria/pull/38)) — Runs `buf lint`, `buf breaking` against `main`, and a generated-code drift check so `go/contracts` never goes stale relative to `contracts/proto/`.
5.  **Docs Link Check** — Verifies that relative markdown links across the repository resolve to existing files.

## Planned (Not Yet Enforced)

The following checks are design goals. They do **not** run in CI today; do not rely on them as gates:

*   **Regression Benchmarks** — Micro-benchmarks on critical hot-paths that fail the build when a PR introduces allocations on designated zero-allocation paths. Today the repository contains a single benchmark (`go/slog-gcp`) and no benchmark job.
*   **OKF Document Integrity Lint** — Schema validation of OKF frontmatter (duplicate UUIDs, dangling `relations`, required fields). Only the link check above exists today.
*   **`release-please` Automation** — Parsing Conventional Commits to draft release PRs and changelogs automatically. Changelog and release management are currently manual.
*   **Publication Validation** — A post-tag dry-run `go list -m` invocation confirming the Go module proxy can resolve the new version.

---

## Release & Version Tagging (Current Practice)

Once a PR is merged into `main`, releases are cut manually:

### 1. Changelogs
`CHANGELOG.md` is maintained by hand using Conventional Commit history as input. (Automation via `release-please` is planned; see above.)

### 2. Path-Prefixed Annotated Tags
Releases are marked utilizing path-prefixed git tags matching the module subdirectory. The tags must be **annotated** to contain metadata:
```bash
# Tagging a retry release
git tag -a go/retry/v0.1.0 -m "Release go/retry v0.1.0"
git push origin go/retry/v0.1.0
```

### 3. Post-Tag Sanity Check
After tagging, manually verify the Go module proxy resolves the new version (e.g. `GOPROXY=proxy.golang.org go list -m github.com/duizendstra/alexandria/go/retry@v0.1.0`).
