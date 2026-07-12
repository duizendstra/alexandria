# gcp (`go/retry/gcp`)

`gcp` is an extension package for `go/retry` providing automated error classification and smart retry behaviors specifically tailored for Google Cloud Platform (GCP) and Google Workspace API client libraries.

## Features

- **Automated Error Classification**: Automatically parses raw errors from `googleapi.Error` and gRPC status codes into retryable/non-retryable decisions.
- **Fail-Fast Evaluation**: Immediately aborts retries on permanent authentication failures (401 Unauthorized, 403 Forbidden, 400 Bad Request) to prevent rate-limit bans or CPU spin waste.
- **Seamless Wrapper Decorators**: Decorates arbitrary Google API execution closures with retry loops automatically.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/retry/gcp
```

## Quick Start

### 1. Wrapping Google API Executions

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/retry"
	"github.com/duizendstra/alexandria/go/retry/gcp"
	"google.golang.org/api/drive/v3"
)

func main() {
	ctx := context.Background()
	policy := retry.Policy{
		MaxAttempts: 5,
		Initial:     200 * time.Millisecond,
	}

	// Create your standard Google Drive Client
	driveService, _ := drive.NewService(ctx)

	var files *drive.FileList
	err := gcp.WithRetry(ctx, policy, func(ctx context.Context) error {
		var dErr error
		files, dErr = driveService.Files.List().PageSize(10).Do()
		return dErr
	})

	if err != nil {
		fmt.Printf("GCP operations failed: %v\n", err)
		return
	}

	fmt.Printf("Fetched %d files successfully!\n", len(files.Files))
}
```

## Retry Classification Table

The evaluator checks both HTTP and gRPC status codes:

| Scenario / Error Code | Action | Reason |
| :--- | :---: | :--- |
| **HTTP 429** / `RESOURCE_EXHAUSTED` | 🔄 **Retry** | Standard GCP Quota limits (retries with delay). |
| **HTTP 5xx** / `INTERNAL`, `UNAVAILABLE` | 🔄 **Retry** | Backend transient server glitches. |
| **HTTP 401** / `UNAUTHENTICATED` | 🛑 **Fail Fast** | Permanent invalid credentials. |
| **HTTP 403** / `PERMISSION_DENIED` | 🛑 **Fail Fast** | Permanent IAM/Domain authorization blocks. |
| **HTTP 404** / `NOT_FOUND` | 🛑 **Fail Fast** | Logical failure (resource missing). |
