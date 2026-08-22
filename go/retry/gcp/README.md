# gcp (`go/retry/gcp`)

`gcp` is an extension package for `go/retry` providing automated error classification and smart retry behaviors specifically tailored for Google Cloud Platform (GCP) and Google Workspace API client libraries.

## Features

- **Automated Error Classification**: Automatically parses raw errors from `googleapi.Error`, gRPC status codes, and `oauth2.RetrieveError` into retryable/non-retryable decisions.
- **Fail-Fast Evaluation**: Immediately aborts retries on permanent authentication failures (401 Unauthorized, 403 Forbidden without quota reason, 400 Bad Request / invalid_grant) to prevent ban loops or CPU spin waste.
- **Smart Quota & `Retry-After` Parsing**: Automatically recognizes 403 `userRateLimitExceeded` / `rateLimitExceeded` / `quotaExceeded` as transient retries, and respects server-provided `Retry-After` headers (both delta-seconds and HTTP-dates).
- **Type-Safe Generics (`WithRetryVal`)**: Direct typed value execution without manual external closure allocations.
- **Customizable Backoff & Observability**: Configurable initial/max backoff ceilings and `OnRetry` telemetry hooks.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/retry/gcp
```

## Quick Start

### 1. Type-Safe API Execution with `WithRetryVal`

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/retry/gcp"
	"google.golang.org/api/drive/v3"
)

func main() {
	ctx := context.Background()

	// Standard Google Drive Client
	driveService, _ := drive.NewService(ctx)

	// Executes with smart retry, Retry-After adherence, and returns typed *drive.FileList
	files, err := gcp.WithRetryVal(ctx, func() (*drive.FileList, error) {
		return driveService.Files.List().PageSize(10).Do()
	},
		gcp.WithMaxAttempts(5),
		gcp.WithInitialBackoff(200*time.Millisecond),
		gcp.WithMaxBackoff(32*time.Second), // Optimal for batch migrations
		gcp.WithOnRetry(func(attempt int, delay time.Duration, err error) {
			fmt.Printf("Retrying Drive API (attempt %d, backoff %v): %v\n", attempt, delay, err)
		}),
	)
	if err != nil {
		fmt.Printf("Drive API operation failed: %v\n", err)
		return
	}

	fmt.Printf("Fetched %d files successfully!\n", len(files.Files))
}
```

### 2. Basic Error-Only Wrapping with `WithRetry`

```go
err := gcp.WithRetry(ctx, func() error {
    return client.Sync(ctx)
}, gcp.WithMaxAttempts(3))
```

## Retry Classification Table

The evaluator checks both HTTP, OAuth2 token endpoints, and gRPC status codes:

| Scenario / Error Code | Action | Reason |
| :--- | :---: | :--- |
| **HTTP 429** / `RESOURCE_EXHAUSTED` | 🔄 **Retry** | Standard GCP Quota limits (retries with exponential delay or `Retry-After`). |
| **HTTP 403** with `rateLimitExceeded` / `userRateLimitExceeded` | 🔄 **Retry** | Google Workspace / Drive quota throttling. |
| **HTTP 5xx** / `INTERNAL`, `UNAVAILABLE` | 🔄 **Retry** | Backend transient server glitches. |
| **OAuth 503 / 429** | 🔄 **Retry** | Token endpoint transient unavailability. |
| **HTTP 401** / `UNAUTHENTICATED` | 🛑 **Fail Fast** | Permanent invalid credentials. |
| **HTTP 403** (Permission Denied / IAM block) | 🛑 **Fail Fast** | Permanent IAM/Domain authorization blocks. |
| **OAuth `invalid_grant` / `unauthorized_client`** | 🛑 **Fail Fast** | Permanent DWD / Service Account key errors. |
| **HTTP 404** / `NOT_FOUND` | 🛑 **Fail Fast** | Logical failure (resource missing). |
