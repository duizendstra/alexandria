# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- **.githooks**: replaced the live hooks with the golden `blueprints/githooks`
  set — `commit-msg` now accepts the `!` breaking-change marker (the previous
  regex rejected the repo's own `feat(async)!:`-style commits) and passes
  git-generated merge/revert/fixup messages; `pre-commit` gains the
  index-based gofmt gate and fine-grained-PAT secret patterns; new
  `pre-push` runs the fail-closed vet/lint/test/build gate across every
  module (multi-module adaptation of the blueprint).
- **.golangci.yml**: pruned 17 dead `depguard` allowlist entries for
  externals no module imports (whatsmeow, libsignal, sqlite, cobra/viper,
  qrterminal, uuid, go-github, …); the allowlist now matches the blueprint
  starter set plus `connectrpc.com/connect` (required by generated
  `go/contracts` code). No lint behavior change for existing code.
- **CONTRIBUTING.md**: the "test all modules" loop now iterates `go.mod`
  files (`find go -name go.mod`) — the previous `go/*/` glob missed all
  nested modules and reached only 5 of 19.
- **README.md**: module index synced with reality — added the missing
  `platform/gcpenv` row, corrected 8 stale version cells (incl.
  `platform/async` → v0.1.0), and fixed the blueprint description to match
  what exists on disk (ko service builds, githooks, golangci profiles).
- **CHANGELOG.md**: restructured — tagged work moved out of `[Unreleased]`
  into dated release-wave sections matching the actual path-prefixed tags.
- **contracts/README.md**: package list synced (34 packages; added the five
  missing: `alx/email`, `alx/postmark`, `common/privacy`,
  `deployment/status`, `timeline/event`) and the `domain/* = v1` vs
  `v1alpha1` versioning convention documented.
- **CI**: `mod-hygiene` now enforces README module-index parity for every
  module and rejects drift between the live `.githooks` and
  `blueprints/githooks` (pre-push exempt as a documented multi-module
  adaptation).

## 2026-07-19 — reliability & governance wave

Released tags: `go/retry/v0.0.4`, `go/retry/gcp/v0.0.4`, `go/slog-gcp/v0.0.3`,
`go/google/v0.0.3`, `go/dataquality/datadiff/v0.0.3`–`v0.0.4`,
`go/observability/audit/v0.0.3`, `go/contracts/v0.0.3`–`v0.0.4`,
`go/platform/apierr/v0.0.3`, `go/platform/async/v0.1.0`,
`go/platform/web/v0.0.2`, `go/platform/gcpenv/v0.0.1`, `go/governance/v0.1.0`,
`go/iac/pulumi/gcpinfra/v0.1.0`, `go/iac/governance/v0.1.0`.

### Added

- **go/platform/gcpenv v0.0.1**: canonical GCP project ID resolver (env vars,
  then metadata service with `GCP_METADATA_DISABLED` bypass); `slog-gcp` and
  `datadiff` adopt it (#50).
- **go/governance v0.1.0**: cloud-agnostic governance domain model — tiered
  plans (Starter/Standard/Enterprise), organizational hierarchy, classification
  dimensions, scope capabilities, and stack export contract. Pure Go, zero
  dependencies (#33).
- **go/iac/pulumi/gcpinfra v0.1.0**: Pulumi building blocks for GCP — folder
  hierarchies (`folders`) and org-level tag keys (`tagkeys`), both
  deletion-protected, consuming validated `go/governance` domain input (#34).
- **go/iac/governance v0.1.0**: configuration-driven Pulumi governance
  blueprint — reads stack config, builds a validated tiered plan, deploys via
  `gcpinfra`, and exports the downstream contract (#35).
- **blueprints/githooks**: golden git hooks for Go repos — Conventional Commits
  validation (git-generated messages pass through), index-based gofmt +
  credential scan on commit, and a fail-closed vet/lint/test/build gate on push
  (#41).
- **blueprints/golangci**: golden golangci-lint profiles — one quality bar in
  two dependency postures: `library` (curated external allowlist, relaxed
  complexity) and `consumer` (stdlib + library modules only, tight complexity)
  (#47).
- **flake.nix**: Nix dev shell pinning `go`, `gotools`, `golangci-lint`,
  `buf`, and `jq` so the documented onboarding works (#39).

### Changed

- **go/platform/async v0.1.0** (breaking): context-aware `Runner` with bounded
  goroutines and TTL janitor (#45); `BatchBuffer` surfaces failed batches
  instead of dropping them (#40).
- **go/retry v0.0.4** (breaking): exhausted `Transport` retries now return an
  error; `Retry-After` honored (#42).
- **go/retry/gcp v0.0.4**: OAuth errors classified structurally before string
  matching (#46).
- **go/observability/audit v0.0.3** (breaking): `Entry.Time` is `time.Time`
  with stable RFC3339 wire format (#49).
- **go/contracts v0.0.4** (breaking): unified conventions; unproven packages
  demoted to `v1alpha1` (#43). Prose "reserved" range comments converted to
  real protobuf `reserved` statements (18 ranges) so tag reuse is rejected at
  the wire level; `contracts` CI job runs `buf lint`, `buf breaking` against
  main, and a generated-code drift check (#38).
- **go/slog-gcp v0.0.3**: `WithProjectID`, `WithEventIDEnabled`,
  `WithTraceResolver`, `WithLabels` setup options; `WithTraceContext` helper
  for async workers; pre-computed trace prefix parsing to reduce hot-path
  allocations.
- **go/google v0.0.3**: uniform retry via transport, single Drive constructor,
  honest `ValidateAccess` (#48); Workspace Drive scanner `WithDriveID` option
  (#31, #32).
- **module hygiene**: standardized `go 1.26` across modules; removed all
  `replace` directives — inter-module requires now pin published tags
  (`platform/web` previously required a non-existent `apierr v0.0.0` and was
  unresolvable for external consumers) (#36, #37); `mod-hygiene` CI job rejects
  committed `replace` directives, `v0.0.0` pins, and modules missing Dependabot
  coverage; Dependabot expanded to all modules + actions; Google, ConnectRPC,
  and Protobuf dependencies upgraded.
- **.golangci.yml**: removed dead rules; config restated as an instance of
  `blueprints/golangci/library.golangci.yml` (#47).
- **documentation**: docs claims aligned with reality — unbuilt machinery
  marked as planned (#44); `contracts/README.md` rewritten around the actual
  Protocol Buffer schemas and Buf workflow; unit tests added for
  `go/slog-gcp/otelgcp` span context extraction.

## 2026-07-12 — initial module harvest

Released tags: `go/retry/v0.0.1`–`v0.0.3`, `go/retry/gcp/v0.0.1`–`v0.0.3`,
`go/google/v0.0.2`, `go/contracts/v0.0.1`–`v0.0.2`,
`go/dataquality/datadiff/v0.0.1`–`v0.0.2`,
`go/observability/audit/v0.0.1`–`v0.0.2`,
`go/discovery/privacyfilter/v0.0.1`–`v0.0.2`,
`go/discovery/search/v0.0.1`–`v0.0.2`, `go/discovery/search/searchtest/v0.0.2`,
`go/platform/apierr/v0.0.1`–`v0.0.2`, `go/platform/async/v0.0.1`–`v0.0.3`,
`go/platform/cache/v0.0.1`, `go/platform/web/v0.0.1`,
`go/slog-gcp/otelgcp/v0.0.2`.

### Added

- **go/retry**: zero-dependency general-purpose exponential backoff retry
  engine with fail-fast `Permanent` error classification and HTTP client
  `Transport` retrier.
- **go/retry/gcp**: extends `go/retry` with comprehensive Google API/GCP error
  classification (rate limits, transient network failures, quota exceeded, and
  OAuth/DWD permanent fail-fast cases).
- **go/dataquality/datadiff**: deep comparison and data validation tool for
  schemas, volume, and metric stats with configurable tolerance limits.
- **go/observability/audit**: production-proven audit logger with structured
  file outputs, automatic file-size rotation, and scorecard readers.
- **go/discovery/privacyfilter**: context-aware scan and redaction filter that
  skips sensitive directory patterns and redacts exposed credentials/tokens.
- **go/discovery/search**: core interfaces and data structures for resilient
  document search, indexing, scoring, and text extraction logic.
- **go/discovery/search/searchtest**: reusable contract tests for
  `search.Index` adapters.
- **go/platform/apierr**: REST API/gRPC error mapping layer with retryable
  classification, unified status responses, and error-unwrapping middleware.
- **go/platform/async**: thread-safe task coordinator for submitting,
  fetching, and pruning background asynchronous tasks.
- **go/platform/cache**: generic, concurrent-safe in-memory TTL cache.
- **go/platform/web**: project-agnostic HTTP server, client, and response
  utilities.
- Root `.golangci.yml` using the standard library lint profile.
- **skills/diffract-review**: agentic Diffract review skill with 9 parallel
  tool-equipped lens agents and CHECK mediator, based on contextvibes/diffract.
- **blueprints/service/.ko.yaml**: golden ko build template for Go Cloud Run
  services (pinned Chainguard static base, reproducible builds).
- **skills/ko-build**: skill for setting up ko container builds with CI/CD
  patterns and troubleshooting.
- **documentation**: SRE-hardened `README.md` files for the core Go modules
  following the standard template pattern.


## [go/slog-gcp/otelgcp/v0.0.1] - 2026-06-28

### Added

- Initial release of the `slog-gcp/otelgcp` module
- OpenTelemetry span context integration as a resolver for `slog-gcp`

## [go/slog-gcp/v0.0.1] - 2026-06-28

### Added

- Initial release of the `slog-gcp` module
- `slog.Handler` decorator with GCP Cloud Logging JSON output
- Cloud Trace header extraction via HTTP middleware
- Cloud Error Reporting integration via `ErrorAttrs()`
- One-call `Setup()` for Cloud Run services
- Test helpers (`SyncBuffer`, `LogEntries`, assertion functions)
- Godoc examples (`ExampleSetup`, `ExampleNewHandler`, `ExampleErrorAttrs`)

## [repository] - 2026-06-28

### Added

- Initial repository structure with 5-concern layout (`go/`, `contracts/`, `skills/`, `blueprints/`, `docs/`)
- Documentation vault following the 8-folder Open Knowledge Format
- GitHub issue templates (bug report, feature request) and PR template
- Git hooks for conventional commit validation and secret scanning
- CI pipeline with dynamic Go module discovery
- Contributor Covenant Code of Conduct
- Security policy with responsible disclosure process
- Dual licensing: Apache-2.0 (code) and CC-BY-4.0 (documentation)
