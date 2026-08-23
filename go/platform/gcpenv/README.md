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

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- `go/slog-gcp` — log resource attribution
- `go/dataquality/datadiff` — BigQuery billing project fallback

### Load-Bearing Promises
1. **Explicit Beats Ambient**: an environment variable wins over the metadata
   server. A caller can always override what the platform reports.
2. **Priority Is Fixed**: the order in which sources are consulted is pinned,
   so the same environment resolves the same way on every run.
3. **Metadata Is A Fallback, And Cached**: the metadata server is consulted
   only when the environment is silent, and the answer is cached rather than
   re-fetched per call.
4. **Unreachable Metadata Degrades**: a non-200 response or an unreachable
   metadata server yields no project rather than an error or a hang, so code
   off-platform behaves predictably.
5. **Metadata Can Be Switched Off**: lookups can be disabled outright, for
   environments where the probe is unwanted.

