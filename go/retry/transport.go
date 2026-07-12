package retry

import (
	"fmt"
	"io"
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
//	transport := retry.Transport(3, func(code int) bool {
//		return code == 429 || code >= 500
//	}, base)
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

func cloneRequest(req *http.Request) *http.Request {
	if req.Body != nil && req.GetBody != nil {
		activeReq := new(http.Request)
		*activeReq = *req

		return activeReq
	}

	return req
}

func rewindRequestBody(req *http.Request) error {
	if req.Body != nil && req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return fmt.Errorf("retry: failed to get request body: %w", err)
		}

		req.Body = body
	}

	return nil
}

func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		const maxDrainBytes = 4096
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBytes))
		_ = resp.Body.Close()
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp    *http.Response
		lastErr error
	)

	activeReq := cloneRequest(req)

	for attempt := range t.maxAttempts {
		if attempt > 0 {
			if err := rewindRequestBody(activeReq); err != nil {
				return nil, err
			}
		}

		resp, lastErr = t.base.RoundTrip(activeReq)

		// Network error — retryable unless marked permanent.
		if lastErr != nil {
			if IsPermanent(lastErr) {
				return nil, fmt.Errorf("retry permanent error: %w", lastErr)
			}

			if attempt < t.maxAttempts-1 {
				if waitErr := t.wait(activeReq, attempt); waitErr != nil {
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
		drainBody(resp)

		if attempt < t.maxAttempts-1 {
			if waitErr := t.wait(activeReq, attempt); waitErr != nil {
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

