package gcp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/retry"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestClassify_gRPCStatus(t *testing.T) {
	// Transient gRPC error status.
	transientStatus := status.New(codes.Unavailable, "connection drops")
	err := Classify(context.Background(), transientStatus.Err(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if retry.IsPermanent(err) {
		t.Error("expected codes.Unavailable to be classified as transient, but got permanent")
	}

	// Permanent gRPC error status.
	permanentStatus := status.New(codes.InvalidArgument, "invalid input value")
	err = Classify(context.Background(), permanentStatus.Err(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !retry.IsPermanent(err) {
		t.Error("expected codes.InvalidArgument to be classified as permanent, but got transient")
	}
}

func newRetrieveError(statusCode int, httpStatus, errorCode string) *oauth2.RetrieveError {
	return &oauth2.RetrieveError{
		Response: &http.Response{
			StatusCode: statusCode,
			Status:     httpStatus,
		},
		Body:      []byte(`{"error":"` + errorCode + `"}`),
		ErrorCode: errorCode,
	}
}

func TestClassify_OAuthRetrieveError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantPermanent bool
	}{
		{
			name:          "invalid_grant is permanent",
			err:           newRetrieveError(http.StatusBadRequest, "400 Bad Request", "invalid_grant"),
			wantPermanent: true,
		},
		{
			name:          "unauthorized_client is permanent",
			err:           newRetrieveError(http.StatusUnauthorized, "401 Unauthorized", "unauthorized_client"),
			wantPermanent: true,
		},
		{
			name:          "invalid_scope is permanent",
			err:           newRetrieveError(http.StatusBadRequest, "400 Bad Request", "invalid_scope"),
			wantPermanent: true,
		},
		{
			name: "invalid_grant wrapped one level",
			err: fmt.Errorf("impersonate: token fetch failed: %w",
				newRetrieveError(http.StatusBadRequest, "400 Bad Request", "invalid_grant")),
			wantPermanent: true,
		},
		{
			name: "invalid_grant wrapped at depth",
			err: fmt.Errorf("drive export: %w",
				fmt.Errorf("impersonate: %w",
					fmt.Errorf("oauth2: cannot fetch token: %w",
						newRetrieveError(http.StatusBadRequest, "400 Bad Request", "invalid_grant")))),
			wantPermanent: true,
		},
		{
			name:          "token endpoint 503 is transient",
			err:           newRetrieveError(http.StatusServiceUnavailable, "503 Service Unavailable", ""),
			wantPermanent: false,
		},
		{
			name:          "token endpoint 429 is transient",
			err:           newRetrieveError(http.StatusTooManyRequests, "429 Too Many Requests", ""),
			wantPermanent: false,
		},
		{
			name: "wrapped 500 with non-RFC error code is transient",
			err: fmt.Errorf("oauth2: cannot fetch token: %w",
				newRetrieveError(http.StatusInternalServerError, "500 Internal Server Error", "server_error")),
			wantPermanent: false,
		},
		{
			name:          "401 without error code is permanent",
			err:           newRetrieveError(http.StatusUnauthorized, "401 Unauthorized", ""),
			wantPermanent: true,
		},
		{
			name:          "403 without error code is permanent",
			err:           newRetrieveError(http.StatusForbidden, "403 Forbidden", ""),
			wantPermanent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(context.Background(), tt.err, 1)
			if got == nil {
				t.Fatal("expected non-nil classified error, got nil")
			}
			if retry.IsPermanent(got) != tt.wantPermanent {
				t.Errorf("Classify() permanent = %v, want %v (err: %v)",
					retry.IsPermanent(got), tt.wantPermanent, got)
			}

			if _, ok := errors.AsType[*oauth2.RetrieveError](got); !ok {
				t.Errorf("expected classified error to preserve *oauth2.RetrieveError, got %v", got)
			}
		})
	}
}

// urlErr wraps cause the way net/http.Client reports every transport
// failure: as a *url.Error, which satisfies net.Error regardless of cause.
func urlErr(cause error) *url.Error {
	return &url.Error{Op: "Get", URL: "https://example.invalid/token", Err: cause}
}

// tcpOpErr wraps cause the way the net package reports a failed socket
// operation on an HTTP connection.
func tcpOpErr(op string, cause error) *net.OpError {
	return &net.OpError{Op: op, Net: "tcp", Err: cause}
}

const opRead = "read"

func TestClassify_URLError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantPermanent bool
	}{
		{
			name: "tls certificate verification is permanent",
			err: urlErr(&tls.CertificateVerificationError{
				Err: x509.UnknownAuthorityError{},
			}),
			wantPermanent: true,
		},
		{
			name: "tls verification wrapped in a token fetch is permanent",
			err: fmt.Errorf("oauth2: cannot fetch token: %w", urlErr(&tls.CertificateVerificationError{
				Err: x509.CertificateInvalidError{Reason: x509.Expired},
			})),
			wantPermanent: true,
		},
		{
			name:          "remote tls alert is permanent",
			err:           urlErr(tcpOpErr("remote error", tls.AlertError(40))),
			wantPermanent: true,
		},
		{
			name:          "context canceled is permanent",
			err:           urlErr(context.Canceled),
			wantPermanent: true,
		},
		{
			name:          "plain client error is permanent",
			err:           urlErr(errors.New("http: server gave HTTP response to HTTPS client")),
			wantPermanent: true,
		},
		{
			name:          "read deadline timeout is transient",
			err:           urlErr(tcpOpErr(opRead, os.ErrDeadlineExceeded)),
			wantPermanent: false,
		},
		{
			name:          "sub-context deadline is transient",
			err:           urlErr(context.DeadlineExceeded),
			wantPermanent: false,
		},
		{
			name:          "connection reset is transient",
			err:           urlErr(tcpOpErr(opRead, &os.SyscallError{Syscall: opRead, Err: syscall.ECONNRESET})),
			wantPermanent: false,
		},
		{
			name:          "connection refused is transient",
			err:           urlErr(tcpOpErr("dial", &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED})),
			wantPermanent: false,
		},
		{
			name:          "temporary resolver failure is transient",
			err:           urlErr(tcpOpErr("dial", &net.DNSError{Err: "server misbehaving", Name: "example.invalid", IsTemporary: true})),
			wantPermanent: false,
		},
		{
			name:          "dropped keep-alive connection is transient",
			err:           urlErr(io.EOF),
			wantPermanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(context.Background(), tt.err, 1)
			if got == nil {
				t.Fatal("expected non-nil classified error, got nil")
			}
			if retry.IsPermanent(got) != tt.wantPermanent {
				t.Errorf("Classify() permanent = %v, want %v (err: %v)",
					retry.IsPermanent(got), tt.wantPermanent, got)
			}
			if !errors.Is(got, tt.err) {
				t.Errorf("expected classified error to preserve %v, got %v", tt.err, got)
			}
		})
	}
}

func TestWithRetry_PermanentURLErrorTLS(t *testing.T) {
	ctx := context.Background()
	calls := 0
	tlsErr := urlErr(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}})

	err := WithRetry(ctx, func() error {
		calls++

		return tlsErr
	}, WithMaxAttempts(5), WithInitialBackoff(time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, tlsErr) {
		t.Errorf("expected wrapped TLS error %v, got %v", tlsErr, err)
	}
	if calls != 1 {
		t.Errorf("expected fail-fast on TLS error after 1 call, got %d", calls)
	}
}

func TestWithRetry_TransientURLErrorTimeout(t *testing.T) {
	ctx := context.Background()
	calls := 0
	timeoutErr := urlErr(tcpOpErr(opRead, os.ErrDeadlineExceeded))

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 2 {
			return timeoutErr
		}

		return nil
	}, WithInitialBackoff(time.Millisecond))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_TransientOAuthRetrieveError(t *testing.T) {
	ctx := context.Background()
	calls := 0

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("oauth2: cannot fetch token: %w",
				newRetrieveError(http.StatusServiceUnavailable, "503 Service Unavailable", ""))
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

func TestWithRetry_PermanentOAuthRetrieveError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	rErr := newRetrieveError(http.StatusBadRequest, "400 Bad Request", "invalid_grant")

	err := WithRetry(ctx, func() error {
		calls++
		return fmt.Errorf("impersonate: %w", rErr)
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, rErr) {
		t.Errorf("expected original RetrieveError preserved, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected fail-fast on invalid_grant after 1 call, got %d", calls)
	}
}

func TestWithRetryVal_Success(t *testing.T) {
	ctx := context.Background()
	type Item struct {
		Name string
	}

	val, err := WithRetryVal(ctx, func() (*Item, error) {
		return &Item{Name: "drive-folder-123"}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil || val.Name != "drive-folder-123" {
		t.Errorf("unexpected value: %+v", val)
	}
}

func TestWithRetryVal_TransientRetry(t *testing.T) {
	ctx := context.Background()
	calls := 0
	apiErr := &googleapi.Error{
		Code:    http.StatusTooManyRequests,
		Message: "rate limit exceeded",
	}

	val, err := WithRetryVal(ctx, func() (int, error) {
		calls++
		if calls < 3 {
			return 0, apiErr
		}
		return 999, nil
	}, WithInitialBackoff(time.Millisecond))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 999 {
		t.Errorf("expected 999, got %d", val)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetryVal_PermanentFailFast(t *testing.T) {
	ctx := context.Background()
	calls := 0
	permErr := &googleapi.Error{
		Code:    http.StatusForbidden,
		Message: "access denied",
	}

	val, err := WithRetryVal(ctx, func() (string, error) {
		calls++
		return "", permErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, permErr) {
		t.Errorf("expected permErr, got %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_RetryAfterHeader(t *testing.T) {
	ctx := context.Background()
	calls := 0
	header := http.Header{}
	header.Set("Retry-After", "0") // 0 seconds so test runs quickly without sleeping.

	apiErr := &googleapi.Error{
		Code:    http.StatusTooManyRequests,
		Message: "rate limited with Retry-After header",
		Header:  header,
	}

	start := time.Now()
	err := WithRetry(ctx, func() error {
		calls++
		if calls < 2 {
			return apiErr
		}
		return nil
	}, WithInitialBackoff(time.Millisecond))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("unexpected long duration: %v", time.Since(start))
	}
}

func TestWithRetry_Options_CustomBackoffAndOnRetry(t *testing.T) {
	ctx := context.Background()
	calls := 0
	retryHookCalls := 0
	var lastDelay time.Duration

	apiErr := &googleapi.Error{
		Code:    http.StatusServiceUnavailable,
		Message: "backend unavailable",
	}

	err := WithRetry(ctx, func() error {
		calls++
		if calls < 3 {
			return apiErr
		}
		return nil
	},
		WithMaxAttempts(5),
		WithInitialBackoff(10*time.Millisecond),
		WithMaxBackoff(50*time.Millisecond),
		WithOnRetry(func(attempt int, delay time.Duration, err error) {
			retryHookCalls++
			lastDelay = delay
		}),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
	if retryHookCalls != 2 {
		t.Errorf("expected OnRetry to be called 2 times, got %d", retryHookCalls)
	}
	if lastDelay <= 0 {
		t.Errorf("expected positive lastDelay, got %v", lastDelay)
	}
}
