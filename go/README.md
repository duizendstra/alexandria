# Go Modules

Independently versioned Go modules for building cloud-native services on GCP.

Each directory is a standalone Go module with its own `go.mod`.

## Module Index

Current versions and import paths live in the
[root module index](../README.md#go-modules), which CI holds in step with
`.release-please-manifest.json`. This table deliberately carries **no version
column** — one gated index is the source of truth, so there is nothing here to
drift out of date.

| Module | Description |
|---|---|
| [retry](retry/) | Exponential backoff/jitter retries and transient HTTP roundtrip retries |
| [retry/gcp](retry/gcp/) | GCP/Google API error classification and retry utilities |
| [slog-gcp](slog-gcp/) | `slog.Handler` decorator for GCP Cloud Logging (trace URLs, error reporting) |
| [slog-gcp/otelgcp](slog-gcp/otelgcp/) | OpenTelemetry trace-context bridge for slog-gcp |
| [google](google/) | Google Workspace authenticator builders, client factories, CRM & Service Usage clients |
| [dataquality/datadiff](dataquality/datadiff/) | Proves two datasets equivalent through layered comparison |
| [observability/audit](observability/audit/) | Structured append-only audit logging |
| [contracts](contracts/) | Compiled Protocol Buffer messages and ConnectRPC services |
| [discovery/privacyfilter](discovery/privacyfilter/) | Content filtering before indexing |
| [discovery/search](discovery/search/) | Core types and port interfaces for the Discovery bounded context |
| [discovery/search/searchtest](discovery/search/searchtest/) | Reusable contract tests for `search.Index` adapters |
| [platform/apierr](platform/apierr/) | Sentinel errors for vendor API interactions |
| [platform/async](platform/async/) | In-memory async task runner with lifecycle states |
| [platform/buildstamp](platform/buildstamp/) | Build provenance stamping and strict pre-run verification (commit, dirty tree, dependency revisions) |
| [platform/cache](platform/cache/) | Generic, concurrent-safe in-memory TTL cache |
| [platform/coordination](platform/coordination/) | Interfaces and primitives for process and workload mutual exclusion and coordination |
| [platform/gcpenv](platform/gcpenv/) | Canonical GCP project ID resolution (env vars + metadata service) |
| [platform/passstore](platform/passstore/) | Deploy-time secret retrieval from the local pass store |
| [platform/procrun](platform/procrun/) | Run external commands under a controlled environment (scrubbed env, fixed PATH, output to file) |
| [platform/retry](platform/retry/) | Exponential backoff/jitter retries and transient HTTP roundtrip retries |
| [platform/runstate](platform/runstate/) | Per-subject run lock and short-lived, fingerprint-bound leases on disk |
| [platform/web](platform/web/) | Project-agnostic HTTP server, client, and response utilities |
| [platform/workflow](platform/workflow/) | Context-aware sequential step procedure engine with panic recovery, skip predicates, and lifecycle hooks |
| [governance](governance/) | Pure-Go governance domain model (scope, tiers, hierarchy, classification) |
| [iac/pulumi/gcpinfra](iac/pulumi/gcpinfra/) | Pulumi adapter packages for Google Cloud infrastructure |
| [iac/pulumi/runner](iac/pulumi/runner/) | Programmatic Pulumi CLI automation engine built on procrun |
| [iac/pulumi/stackref](iac/pulumi/stackref/) | Typed readers for Pulumi stack reference outputs |
| [iac/delivery](iac/delivery/) | Configuration-driven Pulumi blueprint provisioning a CI/CD project (registry, Git connection, build triggers, consumer grants) |
| [iac/finops](iac/finops/) | Configuration-driven Pulumi blueprint provisioning a FinOps project (billing dataset, org budget with alerts) |
| [iac/governance](iac/governance/) | Configuration-driven Pulumi blueprint provisioning GCP governance |
| [iac/identity](iac/identity/) | Configuration-driven Pulumi blueprint provisioning an identity project (secrets, SAs, IAM) |
| [iac/observability](iac/observability/) | Configuration-driven Pulumi blueprint provisioning an observability project (log dataset, org audit-log sink, optional uptime checks + alert channel) |
| [iac/workloads](iac/workloads/) | Configuration-driven Pulumi blueprint provisioning multi-project workload environments with per-concern exports |

## Install

```bash
go get github.com/duizendstra/alexandria/go/<module>@latest
```

## Versioning

Each module is tagged independently: `go/<module>/v0.1.0`
