# Go Modules

Independently versioned Go modules for building cloud-native services on GCP.

Each directory is a standalone Go module with its own `go.mod`.

## Module Index

| Module | Description | Status |
|---|---|---|
| [retry](retry/) | Zero-dependency exponential backoff & HTTP roundtrip retry engine | v0.0.1 |
| [retry/gcp](retry/gcp/) | Google API transient error classification & fail-fast adapter for `go/retry` | v0.0.1 |
| [slog-gcp](slog-gcp/) | GCP Cloud Logging handler for `log/slog` | v0.0.1 |
| [slog-gcp/otelgcp](slog-gcp/otelgcp/) | OpenTelemetry trace context resolver for `slog-gcp` | v0.0.1 |
| [google](google/) | Secure, platform-aware Google Workspace DWD authenticators & Admin/Drive Clients | v0.0.1 |
| [dataquality/datadiff](dataquality/datadiff/) | Schema-based table structures & statistical dataset difference engine | v0.0.1 |
| [observability/audit](observability/audit/) | Dependency-free audit logging domains & size-rotated file adapters | v0.0.1 |
| [discovery/privacyfilter](discovery/privacyfilter/) | Fast, single-pass sensitive document and credential content redactor | v0.0.1 |
| [platform/async](platform/async/) | SRE-hardened panic-resilient bounded asynchronous worker pool | v0.0.1 |

## Install

```bash
go get github.com/duizendstra/alexandria/go/<module>@latest
```

## Versioning

Each module is tagged independently: `go/<module>/v0.1.0`
