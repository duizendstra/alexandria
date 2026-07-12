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
