# go/observability/audit

`go/observability/audit` provides structured, append-only audit logging and file-based rotation adapters to track critical state-mutating actions across microservices.

## Features

- **Structured Append-Only Logs**: Standardized `Entry` schema that records the timestamp, actor, action, and resource.
- **Concurrent FileWriter**: Safe for concurrent write operations across multiple goroutines.
- **Automatic Log Rotation**: Rotates files when they exceed size limits (defaulting to 10 MB).
- **Streaming Scorecard Reader**: Non-allocating JSONL stream-decoder to summarize audit logs.
- **HTTP/RPC Middleware Support**: Identifies actors through the standard `X-Dui-Actor` request header.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/observability/audit
```

## Quick Start

### Logging and Analyzing Audit Actions

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duizendstra/alexandria/go/observability/audit"
	"github.com/duizendstra/alexandria/go/observability/audit/file"
)

func main() {
	ctx := context.Background()
	logPath := "audit.jsonl"

	// Ensure clean-up of example file
	defer os.Remove(logPath)

	// Create size-rotated audit writer
	writer, err := file.NewFileWriter(logPath, file.WithMaxLogSize(1<<20)) // 1 MB limit
	if err != nil {
		log.Fatalf("Failed to initialize audit file writer: %v", err)
	}

	// Write an audit record
	err = writer.Log(ctx, audit.Entry{
		Actor:    "admin-user",
		Action:   "secrets.rotation",
		Resource: "vault/database/credentials",
	})
	if err != nil {
		log.Fatalf("Failed to write log: %v", err)
	}

	// Close to flush changes safely
	_ = writer.Close()

	// Read summary analytics scorecard
	scorecard, err := file.ReadScorecard(logPath)
	if err != nil {
		log.Fatalf("Failed to read scorecard: %v", err)
	}

	fmt.Printf("Total Events: %d, Admin Operations: %d\n", scorecard.Total, scorecard.ByActor["admin-user"])
}
```

## SRE & Performance Hardening details

1. **Thread-Safe Log Rotation**: Checks the file size and executes atomic filesystem rotations inside synchronized blocks to prevent inter-thread race conditions or partial file overwrites.
2. **Streaming Parser Memory Bound**: The `ReadScorecard` analyzer uses streaming chunked JSON decoders rather than loading whole log files into the heap, preserving memory when parsing large files.
3. **Fail-Safe Writer Integrity**: The write pipeline guarantees append-only integrity, ensuring log entries are formatted strictly as single-line JSON records (JSONL), making them safe for generic log aggregators.
