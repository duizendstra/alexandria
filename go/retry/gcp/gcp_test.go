package gcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/retry"
	"google.golang.org/api/googleapi"
)

type mockNetError struct {
	error
	timeout   bool
	temporary bool
}

func (e mockNetError) Timeout() bool {
	return e.timeout
}

func (e mockNetError) Temporary() bool {
	return e.temporary
}

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := WithRetry(ctx, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_PermanentOAuthError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	oauthErr := errors.New("oauth2: cannot fetch token: 401 Unauthorized")

	err := WithRetry(ctx, func() error {
		calls++
		return oauthErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, oauthErr) {
		t.Errorf("expected original OAuth error %v, got %v", oauthErr, err)
	}
	if calls != 1 {
		t.Errorf("expected fail-fast on permanent OAuth error after 1 call, got %d", calls)
	}
}

func TestWithRetry_TransientGoogleAPIError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	apiErr := &googleapi.Error{
		Code:    http.StatusTooManyRequests,
		Message: "rate limit exceeded",
	}

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 3 {
			return apiErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_TransientGCP403QuotaError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	quotaErr := &googleapi.Error{
		Code:    http.StatusForbidden,
		Message: "quota exceeded",
		Errors: []googleapi.ErrorItem{
			{Reason: "quotaExceeded"},
		},
	}

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 2 {
			return quotaErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_PermanentGCP403ForbiddenError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	forbiddenErr := &googleapi.Error{
		Code:    http.StatusForbidden,
		Message: "access denied",
		Errors: []googleapi.ErrorItem{
			{Reason: "accessNotConfigured"},
		},
	}

	err := WithRetry(ctx, func() error {
		calls++
		return forbiddenErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, forbiddenErr) {
		t.Errorf("expected original forbidden error %v, got %v", forbiddenErr, err)
	}
	if !retry.IsPermanent(errors.Unwrap(err)) {
		t.Error("expected returned error to contain permanent wrapper")
	}
	if calls != 1 {
		t.Errorf("expected fail-fast on permanent 403 after 1 call, got %d", calls)
	}
}

func TestWithRetry_TransientNetworkError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	netErr := mockNetError{
		error:     errors.New("timeout connecting"),
		timeout:   true,
		temporary: true,
	}

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 2 {
			return netErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_TransientEOF(t *testing.T) {
	ctx := context.Background()
	calls := 0

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 2 {
			return io.EOF
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_NonAPIPermanentError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	customErr := errors.New("custom non-api failure")

	err := WithRetry(ctx, func() error {
		calls++
		return customErr
	})

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

func TestWithRetry_ContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := WithRetry(ctx, func() error {
		calls++
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if calls != 0 {
		t.Errorf("expected 0 calls since context was pre-canceled, got %d", calls)
	}
}

// Ensure net.Error is verified.
var _ net.Error = mockNetError{}

func TestSetLogger(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(h)

	SetLogger(l)
	defer SetLogger(nil)

	_ = Classify(context.Background(), io.EOF, 1)

	if !strings.Contains(buf.String(), "Transient end-of-file error") {
		t.Errorf("expected log to contain 'Transient end-of-file error', got: %s", buf.String())
	}
}

