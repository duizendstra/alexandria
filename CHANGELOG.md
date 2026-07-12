# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **go/retry**: Zero-dependency general-purpose exponential backoff retry engine with fail-fast `Permanent` error classification and HTTP client `Transport` retrier.
- **go/retry/gcp**: Specialized Go module extending `go/retry` with comprehensive Google API/GCP error classification (handling rate limits, transient network failures, quota exceeded, and OAuth/DWD permanent fail-fast cases).
- Root `.golangci.yml` using the duizendstra-com Library Lint Standard
- **skills/diffract-review**: Agentic Diffract review skill with 9 parallel tool-equipped lens agents and CHECK mediator, based on contextvibes/diffract
- **blueprints/service/.ko.yaml**: Golden ko build template for Go Cloud Run services (pinned Chainguard static base, reproducible builds)
- **skills/ko-build**: Antigravity skill for setting up ko container builds with CI/CD patterns and troubleshooting
- **go/dataquality/datadiff**: Deep comparison and data validation tool for schemas, volume, and metric stats with configurable tolerance limits.
- **go/observability/audit**: Production-proven audit logger with structured file outputs, automatic file-size rotation, and scorecard readers.
- **go/discovery/privacyfilter**: High-security, context-aware scan and redaction filter that skips sensitive directory patterns and redacts exposed credentials/tokens.
- **go/discovery/search**: Core interfaces and data structures for building resilient document search, indexing, scoring, and text extraction logic.
- **go/platform/apierr**: General-purpose REST API/gRPC error mapping layer with retryable classification, unified status responses, and error-unwrapping middleware.
- **go/platform/async**: Thread-safe task coordinator and manager for sub-mitting, fetching, and pruning background asynchronous tasks.
- **documentation**: Generated high-quality, SRE-hardened README.md files for the 9 core Go modules (`go/google`, `go/contracts`, `go/dataquality/datadiff`, `go/observability/audit`, `go/discovery/privacyfilter`, `go/discovery/search`, `go/discovery/search/searchtest`, `go/platform/apierr`, and `go/platform/async`) following the standard template pattern.


### Changed

- **slog-gcp** refactoring and feature additions:
  - Added `WithProjectID`, `WithEventIDEnabled`, `WithTraceResolver`, and `WithLabels` setup options.
  - Added `WithTraceContext` public helper for context propagation in async workers.
  - Added `GCP_METADATA_DISABLED=true` environment variable bypass for metadata query.
  - Optimized trace prefix parsing (pre-computed in handler creation) to reduce hot-path allocations.
- **maintenance**:
  - Upgraded Google, ConnectRPC, and Protobuf dependencies to latest stable/secure versions.
  - Standardized Go modules version requirements to `go 1.26` (libraries minor version rule).
  - Removed all `replace` directives from Go modules for cleaner dependency graph.
  - Added unit tests for `go/slog-gcp/otelgcp` span context extraction.
  - Expanded Dependabot configuration to cover all Go modules with external dependencies + actions.
  - Fixed dead code in `platform/async`, doc comment placement in `platform/apierr`, and doc example in `retry`.
  - Rewrote `contracts/README.md` to document the actual Protocol Buffer schemas and Buf compilation workflow.


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
