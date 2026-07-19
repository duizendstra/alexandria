package retry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testServerHandler struct {
	attempts int
	statuses []int
}

func (h *testServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.attempts < len(h.statuses) {
		code := h.statuses[h.attempts]
		h.attempts++
		w.WriteHeader(code)
		_, _ = w.Write([]byte("body"))

		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("fallback ok"))
}

type errorTransport struct {
	err error
}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, t.err
}

//nolint:dupl // Tests look highly similar but assert different state flows.
func TestTransport_RetryOnStatus(t *testing.T) {
	handler := &testServerHandler{
		statuses: []int{http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusOK},
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	client := &http.Client{
		Transport: Transport(3, shouldRetry, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected request failure: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	if handler.attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", handler.attempts)
	}
}

//nolint:dupl // Tests look highly similar but assert different state flows.
func TestTransport_StopOnNonRetryableStatus(t *testing.T) {
	handler := &testServerHandler{
		statuses: []int{http.StatusServiceUnavailable, http.StatusBadRequest, http.StatusOK},
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	client := &http.Client{
		Transport: Transport(3, shouldRetry, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected request failure: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 BadRequest, got %d", resp.StatusCode)
	}
	if handler.attempts != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", handler.attempts)
	}
}

func TestTransport_PermanentErrorFailFast(t *testing.T) {
	rawErr := errors.New("permanent transport failure")
	base := &errorTransport{err: Permanent(rawErr)}

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	tr := Transport(5, shouldRetry, base)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost", http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tr.RoundTrip(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("expected inner error %v, got %v", rawErr, err)
	}
}

func TestTransport_RetryWithRequestBody(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		if string(body) != "hello retry" {
			t.Errorf("expected body 'hello retry', got %q", string(body))
		}

		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("retry later"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code == http.StatusServiceUnavailable
	}

	client := &http.Client{
		Transport: Transport(3, shouldRetry, nil),
	}

	bodyReader := bytes.NewReader([]byte("hello retry"))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected request failure: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", attempts)
	}
}

func TestTransport_ExhaustedRetriesReturnsResponseAndError(t *testing.T) {
	handler := &testServerHandler{
		statuses: []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable},
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	tr := Transport(2, shouldRetry, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tr.RoundTrip(req)
	if err == nil {
		t.Fatal("expected non-nil error after exhausting retries, got nil")
	}
	if !errors.Is(err, ErrRetriesExceeded) {
		t.Errorf("expected errors.Is(err, ErrRetriesExceeded), got %v", err)
	}
	if resp == nil {
		t.Fatal("expected final retryable response alongside the error, got nil")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected final status 503, got %d", resp.StatusCode)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("failed to read final response body: %v", readErr)
	}
	if string(body) != "body" {
		t.Errorf("expected readable body %q, got %q", "body", string(body))
	}
	if handler.attempts != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", handler.attempts)
	}
}

func TestTransport_ExhaustedRetriesErrorSurfacesThroughClient(t *testing.T) {
	handler := &testServerHandler{
		statuses: []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable},
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	client := &http.Client{
		Transport: Transport(2, shouldRetry, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req) //nolint:bodyclose // net/http discards the response when the transport errors.
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected error from client after exhausting retries, got nil")
	}
	if !errors.Is(err, ErrRetriesExceeded) {
		t.Errorf("expected errors.Is(err, ErrRetriesExceeded) through url.Error, got %v", err)
	}
}

func TestTransport_ExhaustedNetworkErrorWrapsSentinel(t *testing.T) {
	rawErr := errors.New("transient transport failure")
	base := &errorTransport{err: rawErr}

	shouldRetry := func(code int) bool {
		return code >= 500
	}

	tr := Transport(2, shouldRetry, base)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost", http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tr.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
		t.Fatal("expected nil response for exhausted network errors")
	}
	if !errors.Is(err, ErrRetriesExceeded) {
		t.Errorf("expected errors.Is(err, ErrRetriesExceeded), got %v", err)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("expected last network error %v to remain wrapped, got %v", rawErr, err)
	}
}

func TestTransport_HonorsRetryAfterZero(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	shouldRetry := func(code int) bool {
		return code == http.StatusTooManyRequests
	}

	client := &http.Client{
		Transport: Transport(3, shouldRetry, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected request failure: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", attempts)
	}
	// Without Retry-After the two waits take at least Backoff(0)+Backoff(1)
	// = 300ms; with "Retry-After: 0" honored they are immediate.
	if elapsed >= 200*time.Millisecond {
		t.Errorf("expected Retry-After: 0 to skip backoff, request took %v", elapsed)
	}
}

func TestRetryAfterDelay(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		header    string
		wantDelay time.Duration
		wantOK    bool
	}{
		{name: "empty", header: "", wantDelay: 0, wantOK: false},
		{name: "delta seconds", header: "3", wantDelay: 3 * time.Second, wantOK: true},
		{name: "delta seconds padded", header: "  2  ", wantDelay: 2 * time.Second, wantOK: true},
		{name: "delta zero", header: "0", wantDelay: 0, wantOK: true},
		{name: "negative delta rejected", header: "-1", wantDelay: 0, wantOK: false},
		{name: "http date future", header: now.Add(2 * time.Second).Format(http.TimeFormat), wantDelay: 2 * time.Second, wantOK: true},
		{name: "http date past clamps to zero", header: now.Add(-time.Minute).Format(http.TimeFormat), wantDelay: 0, wantOK: true},
		{name: "garbage", header: "soon", wantDelay: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay, ok := retryAfterDelay(tt.header, now)
			if ok != tt.wantOK {
				t.Fatalf("retryAfterDelay(%q) ok = %v, want %v", tt.header, ok, tt.wantOK)
			}
			if delay != tt.wantDelay {
				t.Errorf("retryAfterDelay(%q) delay = %v, want %v", tt.header, delay, tt.wantDelay)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	respWithHeader := func(code int, retryAfter string) *http.Response {
		header := http.Header{}
		if retryAfter != "" {
			header.Set("Retry-After", retryAfter)
		}
		return &http.Response{StatusCode: code, Header: header}
	}

	t.Run("retry-after capped at max backoff", func(t *testing.T) {
		delay := retryDelay(0, respWithHeader(http.StatusTooManyRequests, "3600"))
		if delay != maxBackoff {
			t.Errorf("expected Retry-After 3600s capped at %v, got %v", maxBackoff, delay)
		}
	})

	t.Run("retry-after honored on 503", func(t *testing.T) {
		delay := retryDelay(0, respWithHeader(http.StatusServiceUnavailable, "2"))
		if delay != 2*time.Second {
			t.Errorf("expected 2s from Retry-After, got %v", delay)
		}
	})

	t.Run("retry-after ignored on other statuses", func(t *testing.T) {
		delay := retryDelay(0, respWithHeader(http.StatusInternalServerError, "3600"))
		if delay > time.Second {
			t.Errorf("expected exponential backoff for 500, got %v", delay)
		}
	})

	t.Run("malformed retry-after falls back to backoff", func(t *testing.T) {
		delay := retryDelay(0, respWithHeader(http.StatusTooManyRequests, "whenever"))
		if delay > time.Second {
			t.Errorf("expected exponential backoff fallback, got %v", delay)
		}
	})

	t.Run("nil response falls back to backoff", func(t *testing.T) {
		delay := retryDelay(0, nil)
		if delay > time.Second {
			t.Errorf("expected exponential backoff, got %v", delay)
		}
	})
}
