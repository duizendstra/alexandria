---
uuid: cd2acade-1fc9-4423-896b-07681165d1e1
title: "Adopting Alexandria Modules in Downstream Consumers"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-21T11:20:00Z"
updated_at: "2026-08-21T11:20:00Z"
summary: >
  How-to guide for downstream repositories adopting tagged Alexandria Go modules,
  eliminating vendored/duplicate packages, removing local replace directives, and
  integrating shared platform primitives.
audience: [public]
tags: [ "playbook", "adoption", "modules", "vendoring", "go" ]
relations:
  - target_uuid: "f93b71c5-2fef-48e3-9219-714ad1543083"
    rel_type: "relates_to"
  - target_uuid: "b5ca5a05-41c7-49cb-8cb6-06ea4258c90b"
    rel_type: "relates_to"
---

# 07 — Adopting Alexandria Modules in Downstream Consumers

This playbook guides engineering teams on adopting canonical Alexandria Go modules
(`github.com/duizendstra/alexandria/go/...`) in external consuming repositories. It
covers prerequisites, local de-vendoring, build stamping, error handling semantics,
and common integration pitfalls.

---

## 1. Prerequisites & Clean Workspace Hygiene

### The Local `replace` Directive Trap

When an external consumer tests against a local clone of Alexandria during development,
it is common to add a `replace` directive to `go.mod`:

```
// ANTI-PATTERN: DO NOT COMMIT OR LEAVE IN PRODUCTION CONSUMERS
replace github.com/duizendstra/alexandria/go/google => ../alexandria/go/google
```

**Why this breaks builds:**
- A filesystem `replace` directive silently overrides version resolution and binds the build
  to the state of the local disk directory.
- The local checkout may be uncommitted, on a different branch, or dozens of commits behind
  the published tag.
- Downstream CI/CD and production builds will fail or diverge from local development.

**Rule**: **Always remove filesystem `replace` directives pointing to local Alexandria checkouts
before committing or publishing.**

For local cross-repository development, use an uncommitted `go.work` file at the parent or workspace
root instead of modifying `go.mod`:

```bash
# Create local workspace across consumer and alexandria
cat > go.work << 'EOF'
go 1.26

use (
    .
    ../alexandria/go/google
    ../alexandria/go/platform/buildstamp
)
EOF
```

---

## 2. Dynamic Caller Discovery

Do not assume caller file paths when replacing a vendored package. Use `grep -rl` across the consumer
codebase to discover all direct and indirect references:

```bash
# Find all files importing the vendored package
grep -rl "lib/go/platform/buildstamp" .
grep -rl "lib/go/governance/invariant" .
```

Deriving the caller list dynamically ensures that wrapper packages, CLI commands, and test files
are all updated in a single pass.

---

## 3. Build Provenance Stamping (`-ldflags`)

Alexandria's `go/platform/buildstamp` module provides build provenance extraction (`Get()`),
commit format validation, and dirty tree detection.

### Stamping Rule for Consumers

- If the consumer repository already has its own build-info package (e.g. `cmd/version` or `pkg/buildinfo`)
  that is targeted by its build scripts and deployment pipelines:
  **Keep stamping the consumer-local package in `-ldflags`.**
- `buildstamp` replaces the *parsing, comparison, and verification* logic inside your code — it does
  not require changing your existing build script `-ldflags` targets unless you are adopting
  `buildstamp` as your primary stamp container.

```go
// Example: Consuming buildstamp for validation while retaining consumer stamp variables
stamp := buildstamp.Get()
if err := stamp.RequireClean(); err != nil {
    log.Fatalf("preflight check failed: %v", err)
}
```

---

## 4. Adoption Patterns by Package Category

### Category A: Pure Drop-in Packages (`buildstamp`, `invariant`)
- **Risk Level**: Minimal
- **Approach**: Drop-in replacement.
1. Run `go get github.com/duizendstra/alexandria/go/<module>@v<version>`.
2. Update imports across all files found via `grep -rl`.
3. Delete the local vendored directory (`rm -rf lib/go/...`).
4. Verify with `go test -race ./...`.

### Category B: Infrastructure Primitives (`procrun`, `runstate`)
- **Risk Level**: Low
- **Approach**: Wrap, not rewrite.
- Rather than rewriting existing consumer domain abstractions, adapt internal infrastructure
  interfaces to delegate to Alexandria primitives:
  - `procrun.Runner`: Wraps command execution with isolated PATH lookups, credential environment
    scrubbing (`CLOUDSDK_*`, `GOOGLE_*`), and file-backed log tailing on error.
  - `runstate.Locker` / `runstate.LeaseStore`: Provides exclusive subject locking with `SIGINT`/`SIGTERM`
    signal cleanup and atomic file-based lease storage with clock-skew protection.

### Category C: Google Cloud & Auth Clients (`go/google`, `slog-gcp`)
- **Risk Level**: Medium
- **Approach**: Plan for structured error semantics.
- When replacing bespoke Google Workspace delegation decorators with `auth.DWDValidator.ValidateAccessAs`:
  - Alexandria wraps validation errors using standard `%w` wrapping (`"DWD validation failed: %w"`).
  - Subject mismatches return typed sentinel errors (`auth.ErrSubjectMismatch`).
  - Consumers asserting exact error strings in unit tests should update assertions to use `errors.Is(err, auth.ErrSubjectMismatch)`.

---

## 5. Verification Checklist for Adopters

After migrating to Alexandria packages in a consumer repo, execute this standard verification checklist:

1. **Verify No Lingering Replace Directives**:
   ```bash
   grep "replace.*alexandria" go.mod || echo "go.mod clean"
   ```
2. **Download & Verify Checksums**:
   ```bash
   go mod tidy
   go mod verify
   ```
3. **Run Test Suite with Race Detection**:
   ```bash
   go test -race ./...
   ```
4. **Run Linter**:
   ```bash
   golangci-lint run ./...
   ```
5. **Verify Build & Version Output**:
   ```bash
   go build ./...
   ```
