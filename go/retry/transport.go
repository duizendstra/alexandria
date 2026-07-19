package retry

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrNonRewindableBody is returned when an HTTP request contains a body but no GetBody method,
// preventing the transport from safely rewinding and retrying.
var ErrNonRewindableBody = errors.New("retry: cannot retry request with non-rewindable body")

// ErrRetriesExceeded is returned (wrapped) by the [http.RoundTripper] from
// [Transport] when every attempt has been exhausted without obtaining a
// non-retryable result. Match it with [errors.Is]. When it accompanies a
// non-nil *http.Response, the response carries the final retryable status
// code and an unread body that the caller must close.
var ErrRetriesExceeded = errors.New("retry: retries exceeded")

// Transport returns an [http.RoundTripper] that retries requests when
// shouldRetry returns true for the status code. It waits [Backoff] between
// attempts, honors Retry-After headers on 429 and 503 responses (both
// delta-seconds and HTTP-date forms, capped at the maximum backoff), and
// respects request context cancellation.
//
// The shouldRetry function receives the HTTP status code and returns
// true if the request should be retried. This keeps the retry transport
// decoupled from specific error taxonomies.
//
// Terminal semantics: like [Do], the transport reports failure with a
// non-nil error once all attempts are exhausted. If the final attempt
// produced a retryable HTTP response, RoundTrip returns that response
// together with an error wrapping [ErrRetriesExceeded]; callers must close
// the response body whenever the response is non-nil, even alongside a
// non-nil error. (Note: [net/http.Client] discards a response returned with
// an error, so through a client only the error surfaces.) If the final
// attempt failed with a transport error, RoundTrip returns a nil response
// and an error wrapping both [ErrRetriesExceeded] and the last error.
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
	return req.Clone(req.Context())
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
		const maxDrainBytes = 512 * 1024 // 512KB limit to fully drain bulk API failures safely.
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
			if activeReq.Body != nil && activeReq.Body != http.NoBody && activeReq.GetBody == nil {
				return nil, ErrNonRewindableBody
			}
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
				if waitErr := t.wait(activeReq, attempt, nil); waitErr != nil {
					return nil, waitErr
				}
			}

			continue
		}

		// Success or non-retryable status.
		if !t.shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		// Retryable status on the final attempt: keep the body readable so
		// the caller can inspect it alongside ErrRetriesExceeded below.
		if attempt == t.maxAttempts-1 {
			break
		}

		// Drain and close body before retry to reuse the connection.
		// Headers remain available for Retry-After pacing in wait.
		drainBody(resp)

		if waitErr := t.wait(activeReq, attempt, resp); waitErr != nil {
			return nil, waitErr
		}
	}

	// Attempts exhausted. Return the last response if we have one so the
	// caller can inspect status, headers, and body — but pair it with a
	// non-nil error: a retryable status after exhaustion is a failure, not
	// a success.
	if resp != nil {
		return resp, fmt.Errorf("%w after %d attempts: last status %d", ErrRetriesExceeded, t.maxAttempts, resp.StatusCode)
	}

	return nil, fmt.Errorf("%w after %d attempts: %w", ErrRetriesExceeded, t.maxAttempts, lastErr)
}

// wait sleeps before the next attempt, honoring context cancellation. resp
// is the retryable response from the attempt that just failed (nil when the
// attempt failed with a transport error) and is consulted for Retry-After.
func (t *retryTransport) wait(req *http.Request, attempt int, resp *http.Response) error {
	timer := time.NewTimer(retryDelay(attempt, resp))

	select {
	case <-timer.C:
		return nil
	case <-req.Context().Done():
		timer.Stop()

		return fmt.Errorf("retry: %w", req.Context().Err())
	}
}

// retryDelay returns how long to wait before the next attempt. Servers may
// pace clients explicitly via a Retry-After header on 429 and 503 responses;
// when present and well formed, that value wins, capped at the maximum
// backoff. Otherwise the exponential [Backoff] schedule applies.
func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) {
		if delay, ok := retryAfterDelay(resp.Header.Get("Retry-After"), time.Now()); ok {
			return min(delay, maxBackoff)
		}
	}

	return Backoff(attempt)
}

// retryAfterDelay parses a Retry-After header value in either delta-seconds
// form ("120") or HTTP-date form ("Fri, 31 Dec 1999 23:59:59 GMT"), relative
// to now. It reports false for empty, malformed, or negative values; dates
// in the past yield a zero delay.
func retryAfterDelay(header string, now time.Time) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	if secs, err := strconv.Atoi(header); err == nil {
		if secs < 0 {
			return 0, false
		}

		return time.Duration(secs) * time.Second, true
	}

	if date, err := http.ParseTime(header); err == nil {
		return max(date.Sub(now), 0), true
	}

	return 0, false
}
