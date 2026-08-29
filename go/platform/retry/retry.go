package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

const (
	// backoffBase is the base duration for exponential backoff.
	backoffBase = 100 * time.Millisecond

	// maxBackoff caps the exponential backoff duration before jitter.
	maxBackoff = 5 * time.Second

	// jitterFraction determines the jitter range (1/5 = 20%).
	jitterFraction = 5

	// maxAttemptShift caps the bit shift to prevent integer overflow.
	maxAttemptShift = 30
)

// PermanentError can be implemented by errors to signal that they are permanent
// and should not be retried.
type PermanentError interface {
	Permanent() bool
}

type permanentError struct {
	error
}

func (e permanentError) Unwrap() error {
	return e.error
}

func (e permanentError) Permanent() bool {
	return true
}

// Permanent wraps an error to mark it as permanent so Do or Transport
// will not retry it.
func Permanent(err error) error {
	if err == nil {
		return nil
	}

	return permanentError{err}
}

// IsPermanent checks if an error is wrapped as a permanent error or
// implements a Permanent() bool method returning true.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	var pe PermanentError

	return errors.As(err, &pe) && pe.Permanent()
}

// BackoffWithConfig returns an exponential delay for the given attempt (0-indexed)
// parameterized by base delay and maximum backoff cap, plus 0–20% jitter.
func BackoffWithConfig(attempt int, base, maxCap time.Duration) time.Duration {
	if base <= 0 {
		base = backoffBase
	}
	if maxCap <= 0 {
		maxCap = maxBackoff
	}
	if base >= maxCap {
		b := maxCap
		jit := jitter(b / jitterFraction)

		return b + jit
	}
	shift := min(max(attempt, 0), maxAttemptShift)
	multiplier := int64(1) << uint(shift)
	var b time.Duration
	if multiplier >= int64(maxCap/base)+1 {
		b = maxCap
	} else {
		b = min(time.Duration(multiplier)*base, maxCap)
	}
	jit := jitter(b / jitterFraction)

	return b + jit
}

// Backoff returns an exponential delay for the given attempt (0-indexed).
//
// The delay is 2^attempt × 100ms, capped at 5s, plus 0–20% jitter from
// math/rand/v2. Attempt 0 returns ~100ms, attempt 5 returns ~3.2s,
// attempt 6+ returns ~5s.
func Backoff(attempt int) time.Duration {
	return BackoffWithConfig(attempt, backoffBase, maxBackoff)
}

// Do calls fn up to maxAttempts times. Between failures it waits using
// [Backoff]. It returns immediately if ctx is canceled or if fn returns
// an error marked as permanent via [Permanent] or by implementing [PermanentError].
//
// Terminal semantics: when all attempts are exhausted, Do returns the last
// error from fn unchanged — a non-nil result always means failure, so no
// extra sentinel is needed. The [Transport] counterpart likewise returns a
// non-nil error on exhaustion, but wraps [ErrRetriesExceeded] (optionally
// alongside the final retryable response), because an HTTP response with a
// retryable status would otherwise be indistinguishable from success.
//
//	err := retry.Do(ctx, 3, func() error {
//	    return client.Ping()
//	})
func Do(ctx context.Context, maxAttempts int, fn func() error) error {
	_, err := DoVal(ctx, maxAttempts, func() (struct{}, error) {
		return struct{}{}, fn()
	})

	return err
}

// DoVal calls fn up to maxAttempts times and returns the resulting value and error.
// Between failures it waits using [Backoff]. It returns immediately if ctx is
// canceled or if fn returns an error marked as permanent via [Permanent] or by
// implementing [PermanentError].
//
// On success, DoVal returns the value produced by fn and nil error.
// On failure, DoVal returns the zero value of T and the terminal error.
//
//	val, err := retry.DoVal(ctx, 3, func() (string, error) {
//	    return client.FetchID()
//	})
func DoVal[T any](ctx context.Context, maxAttempts int, fn func() (T, error)) (T, error) {
	var zero T
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var (
		val     T
		lastErr error
		timer   *time.Timer
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for attempt := range maxAttempts {
		val, lastErr = fn()
		if lastErr == nil {
			return val, nil
		}

		if IsPermanent(lastErr) {
			break
		}

		// Don't sleep after the last attempt.
		if attempt == maxAttempts-1 {
			break
		}

		delay := Backoff(attempt)
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(delay)
		}

		select {
		case <-timer.C:
		case <-ctx.Done():
			return zero, fmt.Errorf("retry: %w", ctx.Err())
		}
	}

	return zero, lastErr
}

// jitter returns a random duration in [0, ceiling) using math/rand/v2.
func jitter(ceiling time.Duration) time.Duration {
	if ceiling <= 0 {
		return 0
	}

	return time.Duration(rand.Int64N(int64(ceiling))) // #nosec G404
}
