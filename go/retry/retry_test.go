package retry_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/retry"
)

type customPermErr struct {
	msg string
}

func (e *customPermErr) Error() string   { return e.msg }
func (e *customPermErr) Permanent() bool { return true }

func TestForwarding_Do(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retry.Do(context.Background(), 3, func() error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestForwarding_DoVal(t *testing.T) {
	t.Parallel()

	calls := 0
	val, err := retry.DoVal(context.Background(), 3, func() (string, error) {
		calls++
		if calls < 2 {
			return "", errors.New("transient")
		}
		return "hello", nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %q", val)
	}
}

func TestForwarding_Permanent(t *testing.T) {
	t.Parallel()

	raw := errors.New("fatal")
	wrapped := retry.Permanent(raw)
	if !retry.IsPermanent(wrapped) {
		t.Fatal("expected IsPermanent to return true")
	}

	custom := &customPermErr{msg: "custom"}
	if !retry.IsPermanent(custom) {
		t.Fatal("expected IsPermanent to return true for PermanentError implementation")
	}
}

func TestForwarding_Backoff(t *testing.T) {
	t.Parallel()

	d := retry.Backoff(0)
	if d <= 0 {
		t.Fatalf("expected positive duration, got %v", d)
	}

	cfgD := retry.BackoffWithConfig(0, 50*time.Millisecond, 2*time.Second)
	if cfgD <= 0 {
		t.Fatalf("expected positive duration, got %v", cfgD)
	}
}

func TestForwarding_Transport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: retry.Transport(2, func(code int) bool { return code >= 500 }, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestForwarding_RetryAfterDelay(t *testing.T) {
	t.Parallel()

	delay, ok := retry.RetryAfterDelay("120", time.Now())
	if !ok || delay != 120*time.Second {
		t.Fatalf("expected 120s, got %v, ok=%v", delay, ok)
	}
}
