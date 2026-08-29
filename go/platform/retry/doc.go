// Package retry provides utilities for retrying operations with exponential
// backoff and jitter, as well as transient HTTP roundtrip retries.
//
// # What
//
// A zero-dependency resilience toolkit offering exponential backoff with full
// PCG-based jitter, context-aware execution loops (Do and DoVal), and an
// http.RoundTripper decorator with request body rewind and connection reuse.
//
// # Who
//
// Consumed by HTTP clients, background workers, RPC middleware, and cloud
// adapters across the ecosystem to ensure robust fault tolerance.
//
// # When
//
// Wrap network operations, API interactions, or transient procedural steps
// that may encounter temporary 5xx errors, rate limits (429), or network dropouts.
//
// # Why
//
// Centralizing retry algorithms, timer lifecycle management, and HTTP socket
// draining guarantees uniform resilience behavior while preventing common
// pitfalls like thundering herds, timer leaks, and socket exhaustion.
//
// Domain:  Platform
// Concern: How do we execute resilient retries across operations and HTTP transports?
package retry
