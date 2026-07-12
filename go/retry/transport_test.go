package retry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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
