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
	// Configure standard GCP structured logger
	handler := otelgcp.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

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
