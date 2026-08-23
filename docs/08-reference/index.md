# 08 — Reference

This folder contains look-up material — specifications, API references, and standards that other documents point to.

## What Belongs Here

- **OKF Specification** — The Open Knowledge Format standard used for all documentation in this repository.
- **External API References** — Links and notes on third-party APIs consumed by Alexandria modules.
- **Pointers to Non-Doc Assets** — Where to find the repository's shareable assets that live outside `docs/`.

## Contents

* [Alexandria OKF Profile](okf-profile.md) - How Alexandria uses and extends the Open Knowledge Format (OKF) for its documentation vault.
* [Platform Glossary & Ubiquitous Language](glossary.md) - The canonical lexicon of terms, architectural concepts, and protocols establishing the Ubiquitous Language of Alexandria.

## Engineering Lessons

Short, symptom/cause/rule/how-to-test reference pages distilled from production engineering work, generalized so no client or project specifics are included.

* [Don't Decorate Shared Adapters Used by Evidence-Producing Code](retry-wrapping-shared-adapters.md) - Retry and error-wrapping helpers rewrite the error text they pass through; adding one to a shared adapter method silently changes any downstream string comparison, including verifier and audit-evidence output.
* [Cross-Process Coordination on a Shared Remote Resource](cross-process-locks-shared-remote-resources.md) - A per-process mutex does not protect a remote resource that several processes mutate concurrently; use an advisory file lock around the mutating window, and treat the remote API's conflict response as transient only when the mutation is idempotent.
* [Concurrency-Safe Lazy Initialization: Mutex vs. sync.Once](concurrency-safe-lazy-initialization.md) - sync.Once guarantees its function runs exactly once, not exactly once on success — a first-call failure during lazy initialization is permanent for the life of the process unless a retryable mutex-based pattern is used instead.
* [macOS Shell Pitfalls for Agent-Driven Automation](macos-shell-pitfalls-for-agents.md) - zsh word-splitting and colon modifiers, plus macOS's frozen bash 3.2, make ad hoc multi-step shell commands unreliable for automated workflows on macOS; write such steps as an explicit bash script instead.
* [Go Proxy Tag Lag and Workspace Drift](go-proxy-tag-lag-and-workspace-drift.md) - The Go module proxy's version-list endpoint can lag behind a freshly pushed tag, and an ambient local go.work file masks version-resolution bugs that only appear in CI; verify a tagged version directly and pin GOWORK=off in build scripts that must match CI.

## Repository Assets Outside the Vault

* [`skills/`](../../skills/README.md) - Antigravity AI skills shareable across workspaces (dialectical-review, diffract-review, ko-build, release-review); consumer repos inherit them via `skills.json`.
* [`blueprints/`](../../blueprints/README.md) - Golden configuration templates: the `service/` Go Cloud Run ko build config, the `githooks/` Conventional Commits + quality-gate hook set, and the `golangci/` library/consumer lint profiles.
