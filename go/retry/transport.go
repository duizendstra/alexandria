package retry

import (
	"fmt"
	"net/http"
	"time"
)

// Transport returns an [http.RoundTripper] that retries requests when
// shouldRetry returns true for the status code. It uses [Backoff]
// between attempts and respects request context cancellation.
//
// The shouldRetry function receives the HTTP status code and returns
// true if the request should be retried. This keeps the retry transport
// decoupled from specific error taxonomies.
//
//	transport := retry.Transport(3, apierr.IsRetryableStatus, base)
func Transport(maxAttempts int, shouldRetry func(statusCode int) bool, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	if maxAttempts < 1 {
		maxAttempts = 1
	}

	return &retryTransport{
		base:        base,
		maxAttempts: maxAttempts,
		shouldRetry: shouldRetry,
	}
}

type retryTransport struct {
	base        http.RoundTripper
	maxAttempts int
	shouldRetry func(statusCode int) bool
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp    *http.Response
		lastErr error
	)

	for attempt := range t.maxAttempts {
		resp, lastErr = t.base.RoundTrip(req)

		// Network error — retryable unless marked permanent.
		if lastErr != nil {
			if IsPermanent(lastErr) {
				return nil, fmt.Errorf("retry permanent error: %w", lastErr)
			}

			if attempt < t.maxAttempts-1 {
				if waitErr := t.wait(req, attempt); waitErr != nil {
					return nil, waitErr
				}
			}

			continue
		}

		// Success or non-retryable status.
		if !t.shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		// Drain and close body before retry to reuse the connection.
		_ = resp.Body.Close()

		if attempt < t.maxAttempts-1 {
			if waitErr := t.wait(req, attempt); waitErr != nil {
				return nil, waitErr
			}
		}
	}

	// Return the last response if we have one (even if retryable status),
	// so the caller can inspect the body.
	if resp != nil {
		return resp, nil
	}

	return nil, fmt.Errorf("retry: %w", lastErr)
}

func (t *retryTransport) wait(req *http.Request, attempt int) error {
	delay := Backoff(attempt)
	timer := time.NewTimer(delay)

	select {
	case <-timer.C:
		return nil
	case <-req.Context().Done():
		timer.Stop()

		return fmt.Errorf("retry: %w", req.Context().Err())
	}
}
