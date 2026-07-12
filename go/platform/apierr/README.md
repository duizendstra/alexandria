# go/platform/apierr

`go/platform/apierr` provides a protocol-agnostic error mapping and classification layer designed to handle vendor API interactions with consistent status mappings, retry logic, and memory-bounded debug context.

## Features

- **Semantic Error Sentinels**: Standardized errors (`ErrRateLimited`, `ErrForbidden`, `ErrInvalidInput`, etc.) that map cleanly to both HTTP status codes and gRPC codes.
- **Protocol-Agnostic Conversion**: `FromStatus` (HTTP) and `FromGRPCCode` (gRPC) helpers map raw numeric status codes to standard error sentinels.
- **Transient Error Classification**: `IsRetryable` utility classifies whether a failure is temporary (e.g., rate limits, server timeouts) and should be retried.
- **Memory-Bounded StatusError**: Attaches raw response body snippets (truncated to 4KB) and status codes without exhausting memory.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/apierr
```

## Quick Start

### Handling and Classifying External API Errors

```go
package main

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/platform/apierr"
)

func main() {
	// Simulate an HTTP 429 Rate Limited response with a debug body
	err := apierr.NewStatusError(429, "Rate limit exceeded. Try again in 5s.", apierr.ErrRateLimited)

	// 1. Identify category via errors.Is
	if errors.Is(err, apierr.ErrRateLimited) {
		fmt.Println("Identified: Rate Limited")
	}

	// 2. Extract HTTP status and raw response body via errors.As
	var se *apierr.StatusError
	if errors.As(err, &se) {
		fmt.Printf("HTTP Status Code: %d\nResponse Body Excerpt: %s\n", se.Status, se.Body)
	}

	// 3. Determine retry capability
	if apierr.IsRetryable(err) {
		fmt.Println("Recommendation: Apply backoff and retry the request.")
	}
}
```

## SRE & Performance Hardening details

1. **Memory Bounds Protection**: When instantiating `StatusError`, response bodies are capped to a maximum size of 4096 bytes. This prevents high-throughput services from buffering massive error pages (e.g., standard HTML error pages from proxies) and causing Out-Of-Memory (OOM) situations.
2. **Stateless Operations**: All classification helpers (e.g., `IsRetryable`, `FromStatus`) are pure, stateless functions, eliminating resource contention or lock synchronization inside concurrent request loops.
3. **Dependency-Free Implementation**: Sits at the bottom of the service dependency graph with zero internal dependencies, resolving compilation and diamond-dependency version issues.
