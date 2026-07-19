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
| [`blueprints/`](blueprints/) | **SCAFFOLD** | Project templates — Nix flakes, Go services, Pulumi stacks |
| [`docs/`](docs/) | **KNOW** | Documentation vault — full 8-folder OKF structure |

## Go Modules

| Module | Import Path | Status |
|---|---|---|
| [retry](go/retry/) | `github.com/duizendstra/alexandria/go/retry` | v0.0.1 |
| [retry/gcp](go/retry/gcp/) | `github.com/duizendstra/alexandria/go/retry/gcp` | v0.0.1 |
| [slog-gcp](go/slog-gcp/) | `github.com/duizendstra/alexandria/go/slog-gcp` | v0.0.1 |
| [slog-gcp/otelgcp](go/slog-gcp/otelgcp/) | `github.com/duizendstra/alexandria/go/slog-gcp/otelgcp` | v0.0.1 |
| [google](go/google/) | `github.com/duizendstra/alexandria/go/google` | v0.0.1 |
| [dataquality/datadiff](go/dataquality/datadiff/) | `github.com/duizendstra/alexandria/go/dataquality/datadiff` | v0.0.1 |
| [observability/audit](go/observability/audit/) | `github.com/duizendstra/alexandria/go/observability/audit` | v0.0.1 |
| [contracts](go/contracts/) | `github.com/duizendstra/alexandria/go/contracts` | v0.0.1 |
| [discovery/privacyfilter](go/discovery/privacyfilter/) | `github.com/duizendstra/alexandria/go/discovery/privacyfilter` | v0.0.1 |
| [discovery/search](go/discovery/search/) | `github.com/duizendstra/alexandria/go/discovery/search` | v0.0.1 |
| [platform/apierr](go/platform/apierr/) | `github.com/duizendstra/alexandria/go/platform/apierr` | v0.0.1 |
| [platform/async](go/platform/async/) | `github.com/duizendstra/alexandria/go/platform/async` | v0.0.1 |
| [governance](go/governance/) | `github.com/duizendstra/alexandria/go/governance` | v0.1.0 |


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
