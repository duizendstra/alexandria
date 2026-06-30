# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **go/retry**: Zero-dependency general-purpose exponential backoff retry engine with fail-fast `Permanent` error classification and HTTP client `Transport` retrier.
- **go/retry/gcp**: Specialized sub-module extending `go/retry` with comprehensive Google API/GCP error classification (handling rate limits, transient network failures, quota exceeded, and OAuth/DWD permanent fail-fast cases).
- Root `.golangci.yml` using the duizendstra-com Library Lint Standard
- **skills/diffract-review**: Agentic Diffract review skill with 9 parallel tool-equipped lens agents and CHECK mediator, based on contextvibes/diffract
- **blueprints/service/.ko.yaml**: Golden ko build template for Go Cloud Run services (pinned Chainguard static base, reproducible builds)
- **skills/ko-build**: Antigravity skill for setting up ko container builds with CI/CD patterns and troubleshooting

### Changed

- **slog-gcp** refactoring and feature additions:
  - Added `WithProjectID`, `WithEventIDEnabled`, `WithTraceResolver`, and `WithLabels` setup options.
  - Added `WithTraceContext` public helper for context propagation in async workers.
  - Added `GCP_METADATA_DISABLED=true` environment variable bypass for metadata query.
  - Optimized trace prefix parsing (pre-computed in handler creation) to reduce hot-path allocations.


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
