package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

type customPermanentError struct {
	msg string
}

func (e customPermanentError) Error() string {
	return e.msg
}

func (e customPermanentError) Permanent() bool {
	return true
}

func TestBackoff(t *testing.T) {
	for attempt := range 10 {
		d := Backoff(attempt)
		if d < 100*time.Millisecond {
			t.Errorf("expected backoff for attempt %d to be at least 100ms, got %v", attempt, d)
		}
		if d > 6*time.Second { // 5s cap + 1s max jitter (20%).
			t.Errorf("expected backoff for attempt %d to be capped under 6s, got %v", attempt, d)
		}
	}
}

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := Do(ctx, 3, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 call, got %d", calls)
	}
}

func TestDo_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	calls := 0
	targetCalls := 3
	err := Do(ctx, 5, func() error {
		calls++
		if calls < targetCalls {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != targetCalls {
		t.Errorf("expected %d calls, got %d", targetCalls, calls)
	}
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	calls := 0
	expectedErr := errors.New("persistent transient error")
	err := Do(ctx, 3, func() error {
		calls++
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if calls != 3 {
		t.Errorf("expected exactly 3 calls, got %d", calls)
	}
}

func TestDo_PermanentError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	rawErr := errors.New("fatal problem")
	err := Do(ctx, 5, func() error {
		calls++
		if calls == 2 {
			return Permanent(rawErr)
		}
		return errors.New("transient error")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("expected unwrapped error to be %v, got %v", rawErr, err)
	}
	if !IsPermanent(err) {
		t.Error("expected error to be permanent")
	}
	if calls != 2 {
		t.Errorf("expected execution to stop after 2 calls, got %d", calls)
	}
}

func TestDo_CustomPermanentError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	expectedErr := customPermanentError{msg: "oauth mismatch"}
	err := Do(ctx, 5, func() error {
		calls++
		return expectedErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if !IsPermanent(err) {
		t.Error("expected error to be recognized as permanent")
	}
	if calls != 1 {
		t.Errorf("expected execution to fail-fast after 1 call, got %d", calls)
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	err := Do(ctx, 5, func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return errors.New("transient error")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error to wrap context.Canceled, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected execution to cancel after 2 calls, got %d", calls)
	}
}

type customTransientError struct {
	msg string
}

func (e customTransientError) Error() string {
	return e.msg
}

func (e customTransientError) Permanent() bool {
	return false
}

func TestIsPermanent_NestedOverride(t *testing.T) {
	inner := customTransientError{msg: "temporary transient db failure"}

	// If we wrap it via Permanent, it MUST be recognized as permanent!
	wrapped := Permanent(inner)

	if !IsPermanent(wrapped) {
		t.Error("expected wrapped error to be permanent even if inner implements Permanent() bool false")
	}
}

func TestBackoffWithConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		d := BackoffWithConfig(0, 0, 0)
		if d < 100*time.Millisecond || d > 125*time.Millisecond {
			t.Errorf("unexpected default backoff: %v", d)
		}
	})

	t.Run("custom parameters", func(t *testing.T) {
		base := 500 * time.Millisecond
		maxCap := 30 * time.Second
		d0 := BackoffWithConfig(0, base, maxCap)
		if d0 < 500*time.Millisecond || d0 > 650*time.Millisecond {
			t.Errorf("expected attempt 0 around 500ms, got %v", d0)
		}

		d10 := BackoffWithConfig(10, base, maxCap)
		if d10 < 30*time.Second || d10 > 37*time.Second {
			t.Errorf("expected attempt 10 capped around 30s (+jitter), got %v", d10)
		}
	})
}

func TestDoVal_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0
	val, err := DoVal(ctx, 3, func() (string, error) {
		calls++
		return "hello", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Errorf("expected val 'hello', got %q", val)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoVal_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	calls := 0
	type result struct {
		ID int
	}

	val, err := DoVal(ctx, 5, func() (result, error) {
		calls++
		if calls < 3 {
			return result{}, errors.New("temporary failure")
		}
		return result{ID: 42}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.ID != 42 {
		t.Errorf("expected val ID 42, got %d", val.ID)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoVal_PermanentError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	rawErr := errors.New("unauthorized")

	val, err := DoVal(ctx, 5, func() (int, error) {
		calls++
		if calls == 2 {
			return 0, Permanent(rawErr)
		}
		return 0, errors.New("transient")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("expected %v, got %v", rawErr, err)
	}
	if val != 0 {
		t.Errorf("expected zero value 0, got %d", val)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestDoVal_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	val, err := DoVal(ctx, 5, func() (string, error) {
		calls++
		if calls == 2 {
			cancel()
		}
		return "", errors.New("transient")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error wrapping context.Canceled, got %v", err)
	}
	if val != "" {
		t.Errorf("expected zero string, got %q", val)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}
