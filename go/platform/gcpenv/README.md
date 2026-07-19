# gcpenv

Canonical GCP project ID resolution for the Alexandria ecosystem.

Every module that needs a project ID resolves it here, so the environment
variable priority is identical everywhere:

1. `GCP_PROJECT_ID`, `GOOGLE_CLOUD_PROJECT`, `GCP_PROJECT`, `PROJECT_ID`
   (first non-empty wins)
2. GCE metadata service (Cloud Run, GKE, GCE), unless
   `GCP_METADATA_DISABLED=true`; successful lookups are cached

Returns `""` when undetectable — callers pick their own fallback.

```go
id := gcpenv.ProjectID(ctx)      // env → metadata
id := gcpenv.FromEnv()           // env only (e.g. billing project overrides)
```

Zero dependencies. Tests point `Resolver.MetadataURL` at a fake server.

## Consumers

- `go/slog-gcp` — log resource attribution
- `go/dataquality/datadiff` — BigQuery billing project fallback
