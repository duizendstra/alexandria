# otelgcp (`go/slog-gcp/otelgcp`)

`otelgcp` provides OpenTelemetry context span-to-slog bridges specifically structured for standard Google Cloud Logging and Google Cloud Trace representation.

## Features

- **Trace Correlation**: Automatically links application logs with OpenTelemetry spans in Google Cloud Trace.
- **Low Overhead**: Reads active span contexts directly, preventing expensive formatting or reflection calls.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/slog-gcp/otelgcp
```

## Quick Start

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/duizendstra/alexandria/go/slog-gcp"
	"github.com/duizendstra/alexandria/go/slog-gcp/otelgcp"
	"go.opentelemetry.io/otel"
)

func main() {
	// Configure standard GCP structured logger with OpenTelemetry trace resolver
	inner := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})
	handler := sloggcp.NewHandler(inner, otelgcp.NewResolver(), "my-gcp-project")

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// In your request lifecycle
	tracer := otel.Tracer("my-service")
	ctx, span := tracer.Start(context.Background(), "my-operation")
	defer span.End()

	// Logs written with this context are automatically linked to active Cloud Trace span
	slog.InfoContext(ctx, "Processed operation step successfully")
}
```

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Services already instrumented with OpenTelemetry**: callers wanting their
  existing spans to reach Cloud Logging without a second tracing setup.

### Load-Bearing Promises
1. **One Job, One Constructor**: this package exists to supply a trace resolver
   to `slog-gcp` from an OpenTelemetry context, and nothing else.
2. **The Coupling Is One-Way**: `slog-gcp` does not depend on OpenTelemetry —
   this adapter does. A consumer not using OTel never pulls it in.
3. **Absence Is Not An Error**: a context without an active span yields no
   trace fields rather than a failure, so the resolver is safe to install
   unconditionally.
