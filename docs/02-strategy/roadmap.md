---
uuid: d5e89a5e-df3c-4c48-8df0-e34927f9cb81
title: "Capability Graduation & Roadmap Strategy"
domain: "strategy"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-04T09:00:00Z"
updated_at: "2026-07-19T12:00:00Z"
summary: >
  Defines the lifecycle, quality gates, and graduation roadmap for moving
  custom codebase utilities into standard, shared ecosystem capabilities.
audience: [public]
tags: [ "roadmap", "lifecycle", "strategy" ]
relations:
  - target_uuid: "7c10b0bc-5cb8-4eb4-b99b-efb829623dc1"
    rel_type: "depends_on"
---
# Capability Graduation & Roadmap Strategy

## Overview

Software ecosystems thrive when shared code is curated systematically. speculative abstraction leads to over-engineered, unused utility packages, while isolation leads to duplication and quality decay.

This strategy establishes a rigorous **Graduation Lifecycle** for classifying, vetting, and promoting localized application code into standard, stable ecosystem-wide shared capabilities.

---

## The Capability Lifecycle

Every shared module, contract, or blueprint progresses through four maturity states:

```
[ Seed (Experimental) ] ---> [ Candidate (Review) ] ---> [ Standard (Stable) ] ---> [ Mature (Legacy) ]
```

### 1. Seed (Experimental)
*   **Definition**: A localized utility or prototype addressing an immediate problem in a single application.
*   **Quality Standard**: Code works, but APIs are fluid and documentation is minimal.
*   **Location**: Private application workspaces.

### 2. Candidate (Review)
*   **Definition**: A utility identified as globally useful (needed by 2+ downstream systems) proposed for harvest.
*   **Quality Standard**: Refactored to separate domain logic from adapters. Meets standard Go/Protobuf compilation rules.
*   **Location**: Dedicated feature branch or draft submodule in the shared workspace.

### 3. Standard (Stable)
*   **Definition**: An approved, SRE-hardened, and fully integrated ecosystem capability.
*   **Quality Standard**: High test coverage and zero-allocation hot paths as *target criteria* (see enforcement note below), fully documented with OKF metadata schemas, and automated CI/CD pipeline verification of tests, linting, and module hygiene.
*   **Location**: Merged into `main` and tagged in the shared repository.

### 4. Mature (Legacy)
*   **Definition**: A library or contract that is widely used but has been superseded by a modern pattern.
*   **Quality Standard**: Feature freeze. Security and critical bugs patched, but no active feature development.

---

## Graduation Quality Gates

For a capability to graduate from a **Candidate** to a **Standard**, it must pass the following structural validation criteria. These are **target criteria**: today the Architecture, Resiliency, and Documentation dimensions are checked through code review, and CI enforces tests, `golangci-lint`, and module hygiene. Coverage percentages and allocation benchmarks are **not yet gated in CI** — the repository currently has a single benchmark and no coverage threshold — so the Performance row in particular is verified manually until the planned benchmark gate lands.

| Dimension | Verification Metric | Target Standard | Current Enforcement |
|---|---|---|---|
| **Architecture** | DDD Isolation | 100% separation of core domain models from third-party transport or database libraries. | Code review |
| **Performance** | Benchmarks | Zero-allocations on hot serialization, formatting, or routing paths. | Manual; no CI benchmark gate yet |
| **Resiliency** | Retries & Timeout | All outbound transport adapters must utilize explicit context timeouts and exponential retry backoffs. | Code review |
| **Reproducibility** | Environment | Local dev shell generation via the repository Nix flake. | Nix flake (dev toolchain) |
| **Documentation** | OKF Compliance | Documented inside the `docs/` vault with RFC-compliant, queryable YAML frontmatter. | Review + CI link check |

---

## Multi-Module Versioning & Release Cadence

To guarantee that downstream consumers are insulated from breaking API shifts, the shared workspace employs an independent multi-module versioning scheme:

1.  **Independent Modulating** — Each library folder declares its own `go.mod` file and is versioned independently.
2.  **Semantic Tagging** — Version tags are path-prefixed and annotated:
    `git tag -a go/<module-name>/v<major>.<minor>.<patch> -m "Release message"`
3.  **No Direct Commits** — All promotions are gated behind approved Pull Requests passing automated CI/CD linting, unit testing, and vulnerability sweeps.
