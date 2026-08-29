// Package gcp provides backward-compatible forwarding shims for [github.com/duizendstra/alexandria/go/platform/retry/gcp].
//
// Deprecated: Use [github.com/duizendstra/alexandria/go/platform/retry/gcp] instead.
package gcp

import (
	"context"
	"log/slog"
	"time"

	gcp "github.com/duizendstra/alexandria/go/platform/retry/gcp"
)

// Option configures behavior for WithRetry and WithRetryVal.
//
// Deprecated: Use [gcp.Option] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
type Option = gcp.Option

// SetLogger sets the logger to be used by the package.
//
// Deprecated: Use [gcp.SetLogger] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func SetLogger(l *slog.Logger) {
	gcp.SetLogger(l)
}

// WithMaxAttempts configures the maximum attempts.
//
// Deprecated: Use [gcp.WithMaxAttempts] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithMaxAttempts(attempts int) Option {
	return gcp.WithMaxAttempts(attempts)
}

// WithInitialBackoff configures the initial base delay for exponential backoff.
//
// Deprecated: Use [gcp.WithInitialBackoff] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithInitialBackoff(d time.Duration) Option {
	return gcp.WithInitialBackoff(d)
}

// WithMaxBackoff configures the maximum backoff duration ceiling.
//
// Deprecated: Use [gcp.WithMaxBackoff] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithMaxBackoff(d time.Duration) Option {
	return gcp.WithMaxBackoff(d)
}

// WithOnRetry registers an observability callback invoked before each retry sleep.
//
// Deprecated: Use [gcp.WithOnRetry] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithOnRetry(fn func(attempt int, delay time.Duration, err error)) Option {
	return gcp.WithOnRetry(fn)
}

// WithRetry executes an operation callback function with exponential backoff and GCP-specific error classification.
//
// Deprecated: Use [gcp.WithRetry] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithRetry(ctx context.Context, operation func() error, opts ...Option) error {
	return gcp.WithRetry(ctx, operation, opts...)
}

// WithRetryVal executes an operation callback function with exponential backoff and returns the result.
//
// Deprecated: Use [gcp.WithRetryVal] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func WithRetryVal[T any](ctx context.Context, operation func() (T, error), opts ...Option) (T, error) {
	return gcp.WithRetryVal(ctx, operation, opts...)
}

// Classify determines whether an error should be retried.
//
// Deprecated: Use [gcp.Classify] from [github.com/duizendstra/alexandria/go/platform/retry/gcp].
func Classify(ctx context.Context, err error, attempt int) error {
	return gcp.Classify(ctx, err, attempt)
}
