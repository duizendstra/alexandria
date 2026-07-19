package drive_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/workspace/drive"
	"google.golang.org/api/option"
)

// newRetryingService builds a drive.Service through auth.ResolveClient so all
// calls flow through the retrying transport, pointed at the given test server.
func newRetryingService(t *testing.T, ctx context.Context, endpoint string) *drive.Service {
	t.Helper()

	clientOpts, err := auth.ResolveClient(ctx, nil, auth.WithHTTPClient(&http.Client{}))
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}

	svc, err := drive.New(ctx, drive.Config{}, append(clientOpts, option.WithEndpoint(endpoint))...)
	if err != nil {
		t.Fatalf("drive.New: %v", err)
	}

	return svc
}

// TestService_FetchFile_RetriesTransientErrors proves file downloads survive a
// transient server error thanks to the transport-level retry.
func TestService_FetchFile_RetriesTransientErrors(t *testing.T) {
	ctx := context.Background()

	var calls atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, `{"error":{"code":500,"message":"backend error"}}`, http.StatusInternalServerError)

			return
		}
		_, _ = w.Write([]byte("hello, drive"))
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	content, err := svc.FetchFile(ctx, "file-1", "text/plain", "")
	if err != nil {
		t.Fatalf("expected fetch to survive transient 500, got: %v", err)
	}
	if string(content) != "hello, drive" {
		t.Errorf("unexpected content: %q", content)
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 attempts (1 failure + 1 success), got %d", calls.Load())
	}
}

// TestService_Upload_RetriesTransientErrors proves buffered media uploads are
// rewindable (the SDK supplies GetBody) and therefore retried on transient
// failures.
func TestService_Upload_RetriesTransientErrors(t *testing.T) {
	ctx := context.Background()

	var calls atomic.Int64
	var mu sync.Mutex
	var bodies []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		mu.Lock()
		bodies = append(bodies, buf.String())
		mu.Unlock()

		if n == 1 {
			http.Error(w, `{"error":{"code":503,"message":"unavailable"}}`, http.StatusServiceUnavailable)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"uploaded-1","name":"report.txt"}`))
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	file, err := svc.Upload(ctx, "report.txt", "folder-1", nil, strings.NewReader("payload-bytes"))
	if err != nil {
		t.Fatalf("expected upload to survive transient 503, got: %v", err)
	}
	if file.ID != "uploaded-1" {
		t.Errorf("expected uploaded file ID uploaded-1, got %q", file.ID)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 attempts (1 failure + 1 success), got %d", calls.Load())
	}

	// The replayed request must carry the full media payload again.
	mu.Lock()
	defer mu.Unlock()
	for i, body := range bodies {
		if !strings.Contains(body, "payload-bytes") {
			t.Errorf("attempt %d: expected request body to contain media payload", i+1)
		}
	}
}
