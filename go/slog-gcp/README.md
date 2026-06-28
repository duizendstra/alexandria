# slog-gcp

[![Go Reference](https://pkg.go.dev/badge/github.com/duizendstra/alexandria/go/slog-gcp.svg)](https://pkg.go.dev/github.com/duizendstra/alexandria/go/slog-gcp)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](../../LICENSE)

A [`log/slog`](https://pkg.go.dev/log/slog) handler that outputs structured
JSON in [Google Cloud Logging](https://cloud.google.com/logging/docs/structured-logging) format.

## Features

- Maps slog levels to GCP severity (DEBUG through EMERGENCY)
- Injects `logging.googleapis.com/trace`, `spanId`, and `trace_sampled`
- Auto-generates `event_id` for log correlation
- Cloud Error Reporting integration via `ErrorAttrs()`
- HTTP middleware for Cloud Trace header extraction
- One-call `Setup()` for Cloud Run services
- Single external dependency (`google/uuid`)

## Install

```bash
go get github.com/duizendstra/alexandria/go/slog-gcp@latest
```

## Usage

### Cloud Run (recommended)

```go
package main

import (
    "log/slog"
    "net/http"

    sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

func main() {
    sloggcp.Setup() // Detects GCP env, sets global logger

    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)

    // TraceMiddleware extracts X-Cloud-Trace-Context into context
    http.ListenAndServe(":8080", sloggcp.TraceMiddleware(mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
    slog.InfoContext(r.Context(), "request received",
        "method", r.Method,
        "path", r.URL.Path,
    )
    w.Write([]byte("ok"))
}
```

### Cloud Run Jobs & Batch Workers

For background jobs that process messages (e.g., from Pub/Sub) without an HTTP request:

```go
sloggcp.Setup() // Auto-detects CLOUD_RUN_JOB and enables JSON logging

func processMessage(msg []byte) {
    // Generate a trace context for the message processing lifecycle
    ctx := sloggcp.WithTrace(context.Background())
    slog.InfoContext(ctx, "processing started", "size", len(msg))
}
```

### Generic GCP & OpenTelemetry (e.g. GKE)

If you use OpenTelemetry and deploy to generic GCP environments like GKE, use the `otelgcp` module to automatically extract W3C `traceparent` contexts:

```go
import (
    "log/slog"
    "os"

    sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
    "github.com/duizendstra/alexandria/go/slog-gcp/otelgcp"
)

inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    ReplaceAttr: sloggcp.GCPReplaceAttr,
})

// otelgcp.NewResolver() extracts trace context from OpenTelemetry spans
handler := sloggcp.NewHandler(inner, otelgcp.NewResolver(), "my-gcp-project")
slog.SetDefault(slog.New(handler))
```

### Custom handler

```go
inner := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level:       slog.LevelDebug,
    ReplaceAttr: sloggcp.GCPReplaceAttr,
})
handler := sloggcp.NewHandler(inner, nil, "my-gcp-project")
logger := slog.New(handler)
logger.Info("server started", "port", 8080)
```

## Output

```json
{
  "severity": "INFO",
  "message": "request received",
  "timestamp": "2026-06-28T09:00:00.000Z",
  "logging.googleapis.com/trace": "projects/my-project/traces/abc123",
  "logging.googleapis.com/spanId": "def456",
  "logging.googleapis.com/trace_sampled": true,
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/"
}
```

## Error Reporting

Use `ErrorAttrs()` to generate Cloud Error Reporting fields:

```go
slog.LogAttrs(ctx, slog.LevelError, "database connection failed",
    append(sloggcp.ErrorAttrs(err, sloggcp.ServiceContextFromEnv()), slog.String("db", "firestore"))...,
)
```

This adds `@type`, `serviceContext`, and `error` fields that Cloud
Error Reporting uses for grouping and alerting.

## Testing

Use the built-in test helpers to assert log output:

```go
import sloggcptest "github.com/duizendstra/alexandria/go/slog-gcp/sloggcptest"

func TestMyHandler(t *testing.T) {
    logger, buf := sloggcptest.NewTestLogger(t)

    logger.Info("test message", "key", "value")

    entries := sloggcptest.LogEntries(buf)
    sloggcptest.AssertLogCount(t, entries, 1)
    sloggcptest.AssertLogContains(t, entries, "message", "test message")
}
```

## License

Copyright 2026 Jasper Duizendstra. Licensed under [Apache-2.0](../../LICENSE).
