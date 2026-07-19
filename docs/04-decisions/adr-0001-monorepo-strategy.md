---
title: "ADR-0001: Use a Multi-Module Monorepo"
domain: decisions
type: architecture_decision_record
diataxis_quadrant: explanation
status: accepted
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: All shared Go modules, contracts, documentation, and tooling live in a single repository with independent versioning.
---

# ADR-0001: Use a Multi-Module Monorepo

## Status

Accepted

## Context

The duizendstra ecosystem produces several shared Go modules, API contracts,
documentation, and developer tooling. These artifacts are closely related and
often evolve together, but each Go module must be independently versioned and
importable.

We considered three approaches:

1. **One repo per module** — Maximum isolation but high overhead for cross-cutting
   changes, shared CI, and documentation.
2. **Single-module monorepo** — Simple but forces all consumers to take the same
   version and pulls in unrelated dependencies.
3. **Multi-module monorepo** — Each Go module has its own `go.mod` and version
   tag (`go/<module>/v0.1.0`), while sharing CI, documentation, and governance.

## Decision

We use a **multi-module monorepo** (option 3). The repository is named
"Alexandria" and organized into five top-level concerns:

- `go/` — Independently versioned Go modules
- `contracts/` — API contracts (proto, OpenAPI)
- `skills/` — Antigravity AI skills
- `blueprints/` — Project scaffolding templates
- `docs/` — Documentation vault (8-folder OKF structure)

Each Go module under `go/` is a standalone module with its own `go.mod`. Modules
are tagged with path-prefixed tags (e.g., `go/slog-gcp/v0.1.0`) following the
[Go module reference](https://go.dev/ref/mod#vcs-version).

Version numbers are a deliberate signal: a module stays at `v0.0.x` while its
API shape is experimental, and is bumped to `v0.1.x` only once that API shape
has been validated by at least one real consumer (which is why `go/governance`
sits at v0.1.0 while most siblings are v0.0.x).

A `go.work` file (not committed) enables local cross-module development.

## Consequences

### Easier

- Cross-cutting changes (CI, docs, governance) happen in one place.
- Shared publication checklist and quality gates.
- Single Dependabot configuration covers all modules.
- Contributors only need to clone one repository.

### Harder

- Module tagging requires path-prefixed tags — slightly more complex than simple `v0.1.0`.
- CI must dynamically discover which modules exist and test each independently.
- Risk of accidental cross-module dependencies that bypass `go.mod` isolation.
