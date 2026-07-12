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
updated_at: "2026-07-12T14:30:00Z"
summary: >
  Defines the continuous integration rules, release workflows, and semantic tagging
  standards for multi-module monorepos.
audience: [public]
tags: [ "operations", "ci-cd", "tagging" ]
relations:
  - target_uuid: "b4bc306c-9ba5-4eb8-b99b-efb829623dc1"
    rel_type: "depends_on"
---
# Declarative CI/CD Pipelines & Release Automation

## Operational Objective

To establish a continuous integration and automated release delivery model that validates codebase performance, ensures contract compatibility, and handles multi-module semantic tagging with absolute zero manual intervention.

---

## The Multi-Module Monorepo Release Problem

In a monorepo housing several independent Go modules (such as `/go/retry` and `/go/slog-gcp`), standard repository-wide versioning is a anti-pattern. 
Consumers importing `go/retry` should not have their dependency bumped because of changes in `go/slog-gcp`.

Therefore, we enforce **path-prefixed multi-module semantic versioning**. Each subdirectory acts as an independent release boundary.

---

## Continuous Integration Quality Rules

Our GitHub Actions pipeline executes the following checks on every Pull Request targetting the `main` branch:

```
[ Pull Request ]
       |
       v
+---------------+     +---------------+     +---------------+
| Go Unit Tests | --> | Go Benchmarks | --> | Buf Linter    |
| (test -v)     |     | (bench -mem)  |     | (buf lint)    |
+---------------+     +---------------+     +---------------+
                                                    |
                                                    v
                                            +---------------+
                                            | OKF Doc Lint  |
                                            | (cli docs)    |
                                            +---------------+
```

1.  **Strict Linting & Style** — Runs `golangci-lint` to audit formatting, imports, and static vulnerability patterns. Any linter failure blocks the merge.
2.  **Regression Benchmarks** — Runs micro-benchmarks on critical hot-paths. If a PR introduces memory allocations on designated zero-allocation paths, the build fails.
3.  **Buf Schema Validation** — Audits Protobuf schemas under `contracts/proto/` for backward compatibility breakages using `buf breaking`.
4.  **OKF Document Integrity** — Executes `alexandria-cli docs lint` to audit the documentation directory, blocking PRs on duplicate UUIDs or dangling relations.

---

## Release & Version Tagging Automation

Once a PR is merged into `main`, our release pipeline triggers:

### 1. Automated Changelogs
We utilize `release-please` automation to parse Conventional Commits and draft release PRs. This generates consistent, clean, and automated changelogs.

### 2. Path-Prefixed Annotated Tags
Releases are marked utilizing path-prefixed git tags matching the module subdirectory. The tags must be **annotated** to contain metadata:
```bash
# Tagging a retry release
git tag -a go/retry/v0.1.0 -m "Release go/retry v0.1.0"
git push origin go/retry/v0.1.0
```

### 3. Publication Validation
Once tagged, the pipeline runs a dry-run `go list -m` invocation to ensure the Go module proxy can resolve and index the package cleanly.
