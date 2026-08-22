---
title: Reference
domain: reference
type: index
diataxis_quadrant: reference
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: OKF specification and external API references.
uuid: f83b91ac-d6e1-488d-a4e5-86e7e8d4174c
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-08-22T09:00:00Z"
tags: [ "index", "reference" ]
relations: []
---

# 08 — Reference

This folder contains look-up material — specifications, API references, and standards that other documents point to.

## What Belongs Here

- **OKF Specification** — The Open Knowledge Format standard used for all documentation in this repository.
- **External API References** — Links and notes on third-party APIs consumed by Alexandria modules.
- **Pointers to Non-Doc Assets** — Where to find the repository's shareable assets that live outside `docs/`.

## Contents

| Document | Description |
|---|---|
| [Alexandria OKF Profile](okf-profile.md) | How Alexandria uses and extends the Open Knowledge Format. |
| [Glossary](glossary.md) | The canonical lexicon of terms and Ubiquitous Language establishing architectural patterns. |

## Engineering Lessons

Short, symptom/cause/rule/how-to-test reference pages distilled from
production engineering work, generalized so no client or project specifics
are included.

| Document | Description |
|---|---|
| [Don't Decorate Shared Adapters Used by Evidence-Producing Code](retry-wrapping-shared-adapters.md) | Why retry/error-wrapping helpers belong at the caller, not on a shared adapter method other code depends on for exact error text. |
| [Cross-Process Coordination on a Shared Remote Resource](cross-process-locks-shared-remote-resources.md) | Per-process mutexes don't protect a remote resource mutated by multiple processes; use an advisory file lock and treat the API's conflict response as transient. |
| [Concurrency-Safe Lazy Initialization: Mutex vs. sync.Once](concurrency-safe-lazy-initialization.md) | The poisoned-failure trap of `sync.Once` for lazy init that can fail, and the retryable mutex-based alternative. |
| [macOS Shell Pitfalls for Agent-Driven Automation](macos-shell-pitfalls-for-agents.md) | zsh word-splitting, colon modifiers, and the frozen macOS bash 3.2 — why multi-step shell automation should be a script, not an inline command. |
| [Go Proxy Tag Lag and Workspace Drift](go-proxy-tag-lag-and-workspace-drift.md) | Verifying a freshly tagged module version directly instead of trusting the proxy's version-list endpoint, and pinning `GOWORK=off` in scripts that must match CI. |

## Repository Assets Outside the Vault

| Asset | Description |
|---|---|
| [`skills/`](../../skills/README.md) | Antigravity AI skills shareable across workspaces (dialectical-review, diffract-review, ko-build, release-review); consumer repos inherit them via `skills.json`. |
| [`blueprints/`](../../blueprints/README.md) | Golden configuration templates: the `service/` Go Cloud Run ko build config, the `githooks/` Conventional Commits + quality-gate hook set, and the `golangci/` library/consumer lint profiles. |
