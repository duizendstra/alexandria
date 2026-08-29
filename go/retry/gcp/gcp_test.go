package gcp_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	retry "github.com/duizendstra/alexandria/go/platform/retry"
	gcp "github.com/duizendstra/alexandria/go/retry/gcp"
	"google.golang.org/api/googleapi"
)

func TestForwarding_WithRetry(t *testing.T) {
	t.Parallel()

	calls := 0
	err := gcp.WithRetry(context.Background(), func() error {
		calls++
		if calls < 2 {
			return &googleapi.Error{
				Code:    http.StatusTooManyRequests,
				Message: "rate limit exceeded",
			}
		}
		return nil
	}, gcp.WithMaxAttempts(3), gcp.WithInitialBackoff(time.Millisecond))

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestForwarding_WithRetryVal(t *testing.T) {
	t.Parallel()

	calls := 0
	val, err := gcp.WithRetryVal(context.Background(), func() (string, error) {
		calls++
		if calls < 2 {
			return "", &googleapi.Error{
				Code:    http.StatusTooManyRequests,
				Message: "rate limit exceeded",
			}
		}
		return "result", nil
	}, gcp.WithMaxAttempts(3), gcp.WithInitialBackoff(time.Millisecond))

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if val != "result" {
		t.Fatalf("expected 'result', got %q", val)
	}
}

// TestForwarding_NonAPIErrorIsPermanent locks in the fail-fast guarantee that
// go/retry/gcp made before the relocation: a non-API error is permanent and is
// never retried. The shim must forward that classification unchanged.
func TestForwarding_NonAPIErrorIsPermanent(t *testing.T) {
	t.Parallel()

	customErr := errors.New("custom non-api failure")
	calls := 0
	err := gcp.WithRetry(context.Background(), func() error {
		calls++
		return customErr
	}, gcp.WithMaxAttempts(3), gcp.WithInitialBackoff(time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error %v, got %v", customErr, err)
	}
	if calls != 1 {
		t.Errorf("expected fail-fast on non-api error after 1 call, got %d", calls)
	}
}

func TestForwarding_SetLogger(t *testing.T) {
	t.Parallel()

	gcp.SetLogger(slog.Default())
	gcp.SetLogger(nil)
}

// TestForwarding_Classify checks that the shim forwards the classification
// verdict unchanged in both directions: a 429 stays retryable, a plain error
// comes back marked permanent.
func TestForwarding_Classify(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	if err := gcp.Classify(ctx, nil, 1); err != nil {
		t.Errorf("expected nil for a nil error, got %v", err)
	}

	transient := gcp.Classify(ctx, &googleapi.Error{
		Code:    http.StatusTooManyRequests,
		Message: "rate limit exceeded",
	}, 1)
	if transient == nil {
		t.Fatal("expected an error for a 429, got nil")
	}
	if retry.IsPermanent(transient) {
		t.Error("expected a 429 to stay retryable, got permanent")
	}

	plain := errors.New("custom non-api failure")
	permanent := gcp.Classify(ctx, plain, 1)
	if !retry.IsPermanent(permanent) {
		t.Errorf("expected a non-api error to be permanent, got %v", permanent)
	}
	if !errors.Is(permanent, plain) {
		t.Errorf("expected the original error to survive classification, got %v", permanent)
	}
}

// TestForwarding_WithOnRetry checks that the observability callback registered
// through the shim is invoked with the attempt, delay, and triggering error.
func TestForwarding_WithOnRetry(t *testing.T) {
	t.Parallel()
	var attempts []int
	calls := 0
	err := gcp.WithRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return &googleapi.Error{Code: http.StatusServiceUnavailable, Message: "unavailable"}
		}
		return nil
	},
		gcp.WithMaxAttempts(4),
		gcp.WithInitialBackoff(time.Millisecond),
		gcp.WithOnRetry(func(attempt int, _ time.Duration, err error) {
			if err == nil {
				t.Error("expected the triggering error in the callback, got nil")
			}
			attempts = append(attempts, attempt)
		}),
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(attempts) != 2 {
		t.Errorf("expected 2 retry callbacks, got %d (%v)", len(attempts), attempts)
	}
}

// TestForwarding_WithMaxBackoff checks that the shim forwards the backoff
// ceiling, so a large initial backoff is capped rather than slept through.
func TestForwarding_WithMaxBackoff(t *testing.T) {
	t.Parallel()
	var delays []time.Duration
	calls := 0
	err := gcp.WithRetry(context.Background(), func() error {
		calls++
		if calls < 2 {
			return &googleapi.Error{Code: http.StatusInternalServerError, Message: "boom"}
		}
		return nil
	},
		gcp.WithMaxAttempts(3),
		gcp.WithInitialBackoff(time.Second),
		gcp.WithMaxBackoff(2*time.Millisecond),
		gcp.WithOnRetry(func(_ int, d time.Duration, _ error) {
			delays = append(delays, d)
		}),
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(delays) != 1 {
		t.Fatalf("expected 1 retry callback, got %d", len(delays))
	}
	// Jitter is applied on top of the capped base, so assert the delay landed
	// near the 2ms ceiling rather than near the 1s initial backoff.
	if delays[0] > 100*time.Millisecond {
		t.Errorf("expected the delay capped near 2ms, got %v", delays[0])
	}
}
