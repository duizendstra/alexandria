package retry

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
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
	if errors.As(err, &pe) {
		return pe.Permanent()
	}
	var p permanentError

	return errors.As(err, &p)
}

// Backoff returns an exponential delay for the given attempt (0-indexed).
//
// The delay is 2^attempt × 100ms, capped at 5s, plus 0–20% jitter from
// crypto/rand. Attempt 0 returns ~100ms, attempt 5 returns ~3.2s,
// attempt 6+ returns ~5s.
func Backoff(attempt int) time.Duration {
	shift := min(max(attempt, 0), maxAttemptShift)
	base := min(time.Duration(int64(1)<<uint(shift))*backoffBase, maxBackoff)
	jitter := cryptoJitter(base / jitterFraction)

	return base + jitter
}

// Do calls fn up to maxAttempts times. Between failures it waits using
// [Backoff]. It returns immediately if ctx is canceled or if fn returns
// an error marked as permanent via [Permanent] or by implementing [PermanentError].
//
//	err := retry.Do(ctx, 3, func() error {
//	    return client.Ping()
//	})
func Do(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error

	for attempt := range maxAttempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if IsPermanent(lastErr) {
			break
		}

		// Don't sleep after the last attempt.
		if attempt == maxAttempts-1 {
			break
		}

		delay := Backoff(attempt)
		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()

			return fmt.Errorf("retry: %w", ctx.Err())
		}
	}

	return lastErr
}

// cryptoJitter returns a random duration in [0, ceiling) using crypto/rand.
func cryptoJitter(ceiling time.Duration) time.Duration {
	if ceiling <= 0 {
		return 0
	}

	var b [8]byte

	_, _ = rand.Read(b[:])
	n := binary.LittleEndian.Uint64(b[:])

	return time.Duration(int64(n % uint64(ceiling))) //nolint:gosec // ceiling is int64, so modulo result is guaranteed to fit in int64.
}
