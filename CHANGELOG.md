# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **go/iac/observability**: Configuration-driven Pulumi Observability blueprint — dedicated project with a BigQuery log-analytics dataset and an org-level audit-log sink routed into it (writer identity exported for downstream grants), placement resolved from a governance stack reference.
- **go/iac/finops**: Configuration-driven Pulumi FinOps blueprint — dedicated project with a BigQuery billing-export dataset and an org-scoped budget (threshold alerts, email notification channels), placement resolved from a governance stack reference.
- **go/iac/pulumi/gcpinfra**: Six new building blocks — billing budgets with threshold alerts (`budgets`), BigQuery datasets (`datasets`), org-level log sinks (`logsinks`), Cloud Build v2 Git connections with repo links (`connections`), Artifact Registry repositories with IAM grants (`registries`), and tag-push Cloud Build triggers (`triggers`).
- **go/iac/workloads**: Configuration-driven Pulumi workloads blueprint — one or more projects per environment, each serving one or more concerns with per-concern exports, placement resolved from a governance stack reference, and an optional deploy-access grant for a delivery trigger SA.
- **go/iac/identity**: Configuration-driven Pulumi identity blueprint — dedicated project with Secret Manager secrets (pluggable `SecretResolver`, default `pass`), service accounts, consumer/impersonator IAM, and placement resolved from a governance stack reference.
- **go/iac/pulumi/gcpinfra**: Four new building blocks — GCP projects with API enablement (`projects`), Secret Manager secrets (`secrets`), service accounts (`serviceaccounts`), and project/SA-level IAM member bindings (`iambindings`).
- **go/retry**: Zero-dependency general-purpose exponential backoff retry engine with fail-fast `Permanent` error classification and HTTP client `Transport` retrier.
- **go/retry/gcp**: Specialized Go module extending `go/retry` with comprehensive Google API/GCP error classification (handling rate limits, transient network failures, quota exceeded, and OAuth/DWD permanent fail-fast cases).
- Root `.golangci.yml` using the standard library lint profile
- **skills/diffract-review**: Agentic Diffract review skill with 9 parallel tool-equipped lens agents and CHECK mediator, based on contextvibes/diffract
- **blueprints/service/.ko.yaml**: Golden ko build template for Go Cloud Run services (pinned Chainguard static base, reproducible builds)
- **skills/ko-build**: Antigravity skill for setting up ko container builds with CI/CD patterns and troubleshooting
- **go/dataquality/datadiff**: Deep comparison and data validation tool for schemas, volume, and metric stats with configurable tolerance limits.
- **go/governance**: Cloud-agnostic governance domain model — tiered plans (Starter/Standard/Enterprise), organizational hierarchy, classification dimensions, scope capabilities, and stack export contract. Pure Go, zero dependencies.
- **go/iac/pulumi/gcpinfra**: Pulumi building blocks for GCP — folder hierarchies (`folders`) and org-level tag keys (`tagkeys`), both deletion-protected, consuming validated `go/governance` domain input.
- **go/iac/governance**: Configuration-driven Pulumi governance blueprint — reads stack config, builds a validated tiered plan, deploys via `gcpinfra`, and exports the downstream contract.
- **blueprints/githooks**: Golden git hooks for Go repos — Conventional Commits validation (git-generated messages pass through), index-based gofmt + credential scan on commit, and a fail-closed vet/lint/test/build gate on push.
- **blueprints/golangci**: Golden golangci-lint profiles — one quality bar in two dependency postures: `library` (curated external allowlist, relaxed complexity) and `consumer` (stdlib + library modules only, tight complexity).
- **go/observability/audit**: Production-proven audit logger with structured file outputs, automatic file-size rotation, and scorecard readers.
- **go/discovery/privacyfilter**: High-security, context-aware scan and redaction filter that skips sensitive directory patterns and redacts exposed credentials/tokens.
- **go/discovery/search**: Core interfaces and data structures for building resilient document search, indexing, scoring, and text extraction logic.
- **go/platform/apierr**: General-purpose REST API/gRPC error mapping layer with retryable classification, unified status responses, and error-unwrapping middleware.
- **go/platform/async**: Thread-safe task coordinator and manager for sub-mitting, fetching, and pruning background asynchronous tasks.
- **documentation**: Generated high-quality, SRE-hardened README.md files for the 9 core Go modules (`go/google`, `go/contracts`, `go/dataquality/datadiff`, `go/observability/audit`, `go/discovery/privacyfilter`, `go/discovery/search`, `go/discovery/search/searchtest`, `go/platform/apierr`, and `go/platform/async`) following the standard template pattern.


### Changed

- **.golangci.yml**: removed dead rules — exclusions for seven nonexistent paths, a deny entry for an unused package, and references to files that don't exist; the config is now an instance of `blueprints/golangci/library.golangci.yml`. No behavior change for existing modules.
- **slog-gcp** refactoring and feature additions:
  - Added `WithProjectID`, `WithEventIDEnabled`, `WithTraceResolver`, and `WithLabels` setup options.
  - Added `WithTraceContext` public helper for context propagation in async workers.
  - Added `GCP_METADATA_DISABLED=true` environment variable bypass for metadata query.
  - Optimized trace prefix parsing (pre-computed in handler creation) to reduce hot-path allocations.
- **maintenance**:
  - Upgraded Google, ConnectRPC, and Protobuf dependencies to latest stable/secure versions.
  - Standardized Go modules version requirements to `go 1.26` (libraries minor version rule).
  - Removed all `replace` directives from Go modules for cleaner dependency graph:
    inter-module requires now pin published tags (`platform/web` previously required
    the non-existent `apierr v0.0.0` and was unresolvable for external consumers;
    `google` and `retry/gcp` pinned stale `retry` versions).
  - Added `mod-hygiene` CI job: rejects committed `replace` directives, `v0.0.0`
    pins, and modules missing Dependabot coverage.
  - **contracts**: converted prose "reserved" range comments to real protobuf
    `reserved` statements (18 ranges across the domain protos) so tag reuse is
    rejected at the wire level; added `contracts` CI job running `buf lint`,
    `buf breaking` against main, and a generated-code drift check.
  - Added unit tests for `go/slog-gcp/otelgcp` span context extraction.
  - Expanded Dependabot configuration to cover all Go modules + actions.
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
