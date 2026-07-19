# Alexandria

> The arctic vault. One canonical place for everything I use to build software.

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CI](https://github.com/duizendstra/alexandria/actions/workflows/ci.yml/badge.svg)](https://github.com/duizendstra/alexandria/actions/workflows/ci.yml)

Alexandria owns the shared knowledge, libraries, contracts, and tooling of the
[duizendstra](https://github.com/duizendstra) ecosystem.

## What's Inside

| Directory | Concern | Description |
|---|---|---|
| [`go/`](go/) | **BUILD** | Go modules — each with its own `go.mod`, independently versioned |
| [`contracts/`](contracts/) | **DEFINE** | API contracts — proto, OpenAPI, and schema definitions |
| [`skills/`](skills/) | **TEACH** | Antigravity AI skills — shareable agent instructions |
| [`blueprints/`](blueprints/) | **SCAFFOLD** | Golden configuration templates — ko service builds, git hooks, golangci profiles |
| [`docs/`](docs/) | **KNOW** | Documentation vault — full 8-folder OKF structure |

## Go Modules

| Module | Import Path | Status | Description |
|---|---|---|---|
| [retry](go/retry/) | `github.com/duizendstra/alexandria/go/retry` | v0.0.4 | Exponential backoff/jitter retries and transient HTTP roundtrip retries |
| [retry/gcp](go/retry/gcp/) | `github.com/duizendstra/alexandria/go/retry/gcp` | v0.0.4 | GCP/Google API error classification and retry utilities |
| [slog-gcp](go/slog-gcp/) | `github.com/duizendstra/alexandria/go/slog-gcp` | v0.0.3 | `slog.Handler` decorator for GCP Cloud Logging (trace URLs, error reporting) |
| [slog-gcp/otelgcp](go/slog-gcp/otelgcp/) | `github.com/duizendstra/alexandria/go/slog-gcp/otelgcp` | v0.0.2 | OpenTelemetry trace-context bridge for slog-gcp |
| [google](go/google/) | `github.com/duizendstra/alexandria/go/google` | v0.0.3 | Google Workspace authenticator builders and client factories |
| [dataquality/datadiff](go/dataquality/datadiff/) | `github.com/duizendstra/alexandria/go/dataquality/datadiff` | v0.0.4 | Proves two datasets equivalent through layered comparison |
| [observability/audit](go/observability/audit/) | `github.com/duizendstra/alexandria/go/observability/audit` | v0.0.3 | Structured append-only audit logging |
| [contracts](go/contracts/) | `github.com/duizendstra/alexandria/go/contracts` | v0.0.4 | Compiled Protocol Buffer messages and ConnectRPC services |
| [discovery/privacyfilter](go/discovery/privacyfilter/) | `github.com/duizendstra/alexandria/go/discovery/privacyfilter` | v0.0.2 | Content filtering before indexing |
| [discovery/search](go/discovery/search/) | `github.com/duizendstra/alexandria/go/discovery/search` | v0.0.2 | Core types and port interfaces for the Discovery bounded context |
| [discovery/search/searchtest](go/discovery/search/searchtest/) | `github.com/duizendstra/alexandria/go/discovery/search/searchtest` | v0.0.2 | Reusable contract tests for `search.Index` adapters |
| [platform/apierr](go/platform/apierr/) | `github.com/duizendstra/alexandria/go/platform/apierr` | v0.0.3 | Sentinel errors for vendor API interactions |
| [platform/async](go/platform/async/) | `github.com/duizendstra/alexandria/go/platform/async` | v0.1.0 | In-memory async task runner with lifecycle states |
| [platform/cache](go/platform/cache/) | `github.com/duizendstra/alexandria/go/platform/cache` | v0.0.1 | Generic, concurrent-safe in-memory TTL cache |
| [platform/gcpenv](go/platform/gcpenv/) | `github.com/duizendstra/alexandria/go/platform/gcpenv` | v0.0.1 | Canonical GCP project ID resolution (env vars + metadata service) |
| [platform/web](go/platform/web/) | `github.com/duizendstra/alexandria/go/platform/web` | v0.0.2 | Project-agnostic HTTP server, client, and response utilities |
| [governance](go/governance/) | `github.com/duizendstra/alexandria/go/governance` | v0.2.0 | Pure-Go governance domain model (scope, tiers, hierarchy, classification) |
| [iac/pulumi/gcpinfra](go/iac/pulumi/gcpinfra/) | `github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra` | v0.1.1 | Pulumi adapter packages for Google Cloud infrastructure |
| [iac/governance](go/iac/governance/) | `github.com/duizendstra/alexandria/go/iac/governance` | v0.1.1 | Configuration-driven Pulumi blueprint provisioning GCP governance |

Version numbers signal maturity: `v0.0.x` modules are experimental; `v0.1.x`
means the API shape has been validated by at least one real consumer (see
[ADR-0001](docs/04-decisions/adr-0001-monorepo-strategy.md)).


## Documentation

The `docs/` directory is an [Open Knowledge Format (OKF)](https://okf.md) bundle
with an 8-domain structure designed for both human and agentic consumption.
See the [Alexandria OKF Profile](docs/08-reference/okf-profile.md) for details.

| # | Domain | Verb |
|---|---|---|
| 01 | governance | GOVERN |
| 02 | strategy | STRATEGIZE |
| 03 | architecture | DESIGN |
| 04 | decisions | DECIDE |
| 05 | security | PROTECT |
| 06 | operations | RUN |
| 07 | playbooks | GUIDE |
| 08 | reference | LOOK UP |

## License

Code (`go/`, `contracts/`, `.github/`, `.githooks/`) is licensed under
[Apache-2.0](LICENSE).

Documentation, skills, and blueprints (`docs/`, `skills/`, `blueprints/`) are
licensed under [CC-BY-4.0](docs/LICENSE).

Copyright 2026 Jasper Duizendstra.
