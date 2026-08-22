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
2.  **Per-Module Format, Vet, Test & Lint** — For each module: `gofmt -l .` (must print nothing), `go vet ./...`, `GOOS=windows GOARCH=amd64 go vet ./...` (the library is consumed on macOS and native Windows), `go test -race -count=1 -coverprofile=coverage.out ./...`, `govulncheck`, and `golangci-lint`. Any failure blocks the merge. Coverage ratchet is enforced against calibrated module floors in `.github/coverage-baselines.json`.
3.  **Module Hygiene** (`mod-hygiene`) — Rejects committed `replace` directives, unresolvable `v0.0.0` pins, missing Dependabot coverage, missing coverage baselines, and missing `release-please` manifest/config entries or README version mismatches.
4.  **Contracts** (`contracts`) — Runs `buf lint`, `buf breaking` against `main`, and a generated-code drift check so `go/contracts` never goes stale relative to `contracts/proto/`.
5.  **Docs Link Check** — Verifies that relative markdown links across the repository resolve to existing files.
6.  **Release Please Automation** (`.github/workflows/release.yml`) — On pushes to `main`, executes Google's `release-please` in manifest mode (`.release-please-config.json` and `.release-please-manifest.json`), maintaining release PRs with changelogs and cutting path-prefixed tags (`go/<module>/vX.Y.Z`) on merge.

## Planned (Not Yet Enforced)

The following checks are design goals. They do **not** run in CI today; do not rely on them as gates:

*   **Regression Benchmarks** — Micro-benchmarks on critical hot-paths that fail the build when a PR introduces allocations on designated zero-allocation paths. Today the repository contains a single benchmark (`go/slog-gcp`) and no benchmark job.
*   **OKF Document Integrity Lint** — Schema validation of OKF frontmatter (duplicate UUIDs, dangling `relations`, required fields). Only the link check above exists today.
*   **Publication Validation** — A post-tag dry-run `go list -m` invocation confirming the Go module proxy can resolve the new version.

---

## Release & Version Tagging (Automated Practice)

Releases are managed via Conventional Commits and automated via `release-please`:

### 1. Conventional Commits & Release PRs
`release-please` scans commits landed on `main`. For every affected module directory, it maintains a dedicated Release PR with an updated semver bump and per-module changelog.

### 2. Path-Prefixed Annotated Tags
Merging a Release PR automatically generates GitHub releases and path-prefixed git tags (`go/<module>/vX.Y.Z`) matching the module subdirectory.

### 3. Post-Tag Sanity Check
After tagging, verify the Go module proxy resolves the new version:
```bash
GOPROXY=proxy.golang.org go list -m github.com/duizendstra/alexandria/go/retry@v0.1.0
```

