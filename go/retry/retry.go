// Package retry provides backward-compatible forwarding shims for [github.com/duizendstra/alexandria/go/platform/retry].
//
// Deprecated: Use [github.com/duizendstra/alexandria/go/platform/retry] instead.
package retry

import (
	"context"
	"net/http"
	"time"

	"github.com/duizendstra/alexandria/go/platform/retry"
)

// Sentinel errors forwarded from the platform/retry package.
var (
	// Deprecated: Use [retry.ErrNonRewindableBody] from [github.com/duizendstra/alexandria/go/platform/retry].
	ErrNonRewindableBody = retry.ErrNonRewindableBody

	// Deprecated: Use [retry.ErrRetriesExceeded] from [github.com/duizendstra/alexandria/go/platform/retry].
	ErrRetriesExceeded = retry.ErrRetriesExceeded
)

// PermanentError can be implemented by errors to signal that they are permanent and should not be retried.
//
// Deprecated: Use [retry.PermanentError] from [github.com/duizendstra/alexandria/go/platform/retry].
type PermanentError = retry.PermanentError

// Permanent wraps an error to mark it as permanent so Do or Transport will not retry it.
//
// Deprecated: Use [retry.Permanent] from [github.com/duizendstra/alexandria/go/platform/retry].
func Permanent(err error) error {
	return retry.Permanent(err)
}

// IsPermanent checks if an error is wrapped as a permanent error or implements PermanentError.
//
// Deprecated: Use [retry.IsPermanent] from [github.com/duizendstra/alexandria/go/platform/retry].
func IsPermanent(err error) bool {
	return retry.IsPermanent(err)
}

// Backoff returns an exponential backoff duration with randomized jitter for the given attempt.
//
// Deprecated: Use [retry.Backoff] from [github.com/duizendstra/alexandria/go/platform/retry].
func Backoff(attempt int) time.Duration {
	return retry.Backoff(attempt)
}

// BackoffWithConfig returns an exponential delay for the given attempt with custom parameters.
//
// Deprecated: Use [retry.BackoffWithConfig] from [github.com/duizendstra/alexandria/go/platform/retry].
func BackoffWithConfig(attempt int, base, maxCap time.Duration) time.Duration {
	return retry.BackoffWithConfig(attempt, base, maxCap)
}

// Do calls fn up to maxAttempts times, waiting between failures using exponential backoff.
//
// Deprecated: Use [retry.Do] from [github.com/duizendstra/alexandria/go/platform/retry].
func Do(ctx context.Context, maxAttempts int, fn func() error) error {
	return retry.Do(ctx, maxAttempts, fn)
}

// DoVal calls fn up to maxAttempts times and returns the resulting value and error.
//
// Deprecated: Use [retry.DoVal] from [github.com/duizendstra/alexandria/go/platform/retry].
func DoVal[T any](ctx context.Context, maxAttempts int, fn func() (T, error)) (T, error) {
	return retry.DoVal(ctx, maxAttempts, fn)
}

// Transport returns an http.RoundTripper that retries requests.
//
// Deprecated: Use [retry.Transport] from [github.com/duizendstra/alexandria/go/platform/retry].
func Transport(maxAttempts int, shouldRetry func(statusCode int) bool, base http.RoundTripper) http.RoundTripper {
	return retry.Transport(maxAttempts, shouldRetry, base)
}

// RetryAfterDelay parses a Retry-After header value in either delta-seconds form or HTTP-date form.
//
// Deprecated: Use [retry.RetryAfterDelay] from [github.com/duizendstra/alexandria/go/platform/retry].
func RetryAfterDelay(header string, now time.Time) (time.Duration, bool) {
	return retry.RetryAfterDelay(header, now)
}
