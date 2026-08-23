# go/retry

`go/retry` provides robust, thread-safe backoff and retry policies for cloud-native Go microservices.

## Features

- **Exponential Backoff**: Exponential interval scaling with lock-free randomized jitter (backed by `math/rand/v2`).
- **HTTP Transport Wrapper**: A transparent `http.RoundTripper` decorator that automatically retries failed requests.
- **Request Body Rewinding**: Fully supports retrying requests with payloads (`POST`/`PUT`) by safely cloning and resetting request body streams.
- **Keep-Alive Socket Draining**: Pre-consumes up to 4KB of failed response bodies before closing them to preserve and reuse TCP keep-alive sockets.
- **GCP Optimization Integration**: Seamlessly integrates with the `gcp` sub-package for automatic error classification of Google Cloud and Google Workspace API failures.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/retry
```

## Quick Start

### 1. Basic Function Execution Retry

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/retry"
)

func main() {
	ctx := context.Background()

	// Retry the operation up to 5 times using exponential backoff + jitter
	err := retry.Do(ctx, 5, func() error {
		// Your custom transient network or database operation here
		return errors.New("temporary transient error")
	})

	if err != nil {
		fmt.Printf("Operation failed after retries: %v\n", err)
	}
}
```

### 2. Type-Safe Function Execution with Return Values (`DoVal`)

```go
package main

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/retry"
)

type User struct {
	ID   string
	Name string
}

func main() {
	ctx := context.Background()

	user, err := retry.DoVal(ctx, 3, func() (*User, error) {
		// Clean type-safe return without external closure variable allocations
		return &User{ID: "usr-123", Name: "Alex"}, nil
	})
	if err != nil {
		fmt.Printf("Failed to fetch user: %v\n", err)
		return
	}

	fmt.Printf("Fetched user: %s (%s)\n", user.Name, user.ID)
}
```

### 3. HTTP Client Decorator with Connection Protection

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/duizendstra/alexandria/go/retry"
)

func main() {
	shouldRetry := func(code int) bool {
		return code == 429 || code >= 500
	}

	// Create an HTTP client decorated with auto-retries (3 attempts) and TCP connection draining
	client := &http.Client{
		Transport: retry.Transport(3, shouldRetry, http.DefaultTransport),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
}
```

## SRE & Performance Hardening details

1. **Keep-Alive Protection**: Calling `Close()` on a response body before its bytes are fully consumed tears down the underlying TCP connection. The `retry.Transport` automatically copies and discards trailing bytes (up to 4KB) prior to closure to permit connection pooling.
2. **Payload Jitter Speed**: Retry interval randomization is backed by `math/rand/v2`'s PCG-based lock-free random generator, avoiding global CPU mutex contentions in concurrent environments.
3. **Stream Resets**: `retry.Transport` clones request payload bodies before each try via `req.GetBody()`, resolving "empty request body" failures on subsequent attempts of `POST`/`PUT` requests.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **API clients over flaky transports**: callers wrapping an outbound HTTP or
  RPC call that fails for reasons that go away on their own.
- **Long-running migrators and pipelines**: sequential work where one transient
  failure must not abort the run, but a permanent one must abort it at once.

### Load-Bearing Promises
1. **Permanent Means One Attempt**: an error marked with `Permanent` (or
   classified permanent through a nested wrap) is returned after a single
   attempt, with no backoff slept.
2. **Nested Classification Wins**: `IsPermanent` sees through wrapping, so a
   permanent error stays permanent no matter how many layers wrap it.
3. **Context Ends It**: cancellation or deadline expiry stops progression
   before the next attempt rather than after it.
4. **Exhaustion Returns Both**: when `Transport` runs out of attempts it
   returns the last response *and* an error, and that error surfaces through a
   plain `http.Client` rather than being swallowed by it.
5. **Bodies Replay**: request payloads are cloned via `req.GetBody()` before
   each attempt, so a retried `POST` or `PUT` is not sent with an empty body.
6. **`Retry-After` Is Honoured**: a `Retry-After` on the response drives the
   next delay, including a zero value.
7. **Exhausted Network Errors Wrap A Sentinel**: transport-level exhaustion
   returns an error that `errors.Is` matches, so callers can tell it apart
   from the origin's own failure.
