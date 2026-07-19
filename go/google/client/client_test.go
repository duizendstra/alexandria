package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/client"
	workspacedrive "github.com/duizendstra/alexandria/go/google/workspace/drive"
	google_drive "google.golang.org/api/drive/v3"
)

func TestNewServices_InjectClient(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{}

	t.Run("NewDriveService", func(t *testing.T) {
		srv, err := client.NewDriveService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil drive service")
		}
	})

	t.Run("NewAdminService", func(t *testing.T) {
		srv, err := client.NewAdminService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil admin service")
		}
	})

	t.Run("NewReportsService", func(t *testing.T) {
		srv, err := client.NewReportsService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil reports service")
		}
	})
}

// TestNewDriveService_ScannerSurvivesTransient429 proves that a transient
// rate-limit response in the middle of a paginated crawl no longer aborts the
// scan: the retrying transport injected by auth.ResolveClient replays the
// failed page request transparently.
func TestNewDriveService_ScannerSurvivesTransient429(t *testing.T) {
	ctx := context.Background()

	var calls atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("pageToken") {
		case "":
			_, _ = w.Write([]byte(`{"nextPageToken":"page-2","files":[{"id":"file-1"}]}`))
		case "page-2":
			// Fail the second page exactly once with a transient rate limit.
			if calls.Load() == 2 {
				http.Error(w, `{"error":{"code":429,"message":"rate limited"}}`, http.StatusTooManyRequests)

				return
			}
			_, _ = w.Write([]byte(`{"files":[{"id":"file-2"}]}`))
		}
	}))
	defer ts.Close()

	srv, err := client.NewDriveService(ctx, auth.WithHTTPClient(&http.Client{}))
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}
	srv.BasePath = ts.URL + "/"

	var got []string
	scanner := workspacedrive.NewScanner(workspacedrive.WithPageSize(1))
	err = scanner.Scan(ctx, srv, func(f *google_drive.File) error {
		got = append(got, f.Id)

		return nil
	})
	if err != nil {
		t.Fatalf("expected scan to survive transient 429, got: %v", err)
	}

	if len(got) != 2 || got[0] != "file-1" || got[1] != "file-2" {
		t.Errorf("expected files [file-1 file-2], got %v", got)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 requests (page 1, failed page 2, retried page 2), got %d", calls.Load())
	}
}

// TestConstructionPathsEquivalent verifies that the factory in this package
// and the canonical workspace/drive constructor produce equivalent raw Drive
// clients for the same resolved options.
func TestConstructionPathsEquivalent(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{}

	viaFactory, err := client.NewDriveService(ctx, auth.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("NewDriveService: %v", err)
	}

	clientOpts, err := auth.ResolveClient(ctx, []string{google_drive.DriveMetadataReadonlyScope}, auth.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}

	viaWrapper, err := workspacedrive.New(ctx, workspacedrive.Config{}, clientOpts...)
	if err != nil {
		t.Fatalf("workspacedrive.New: %v", err)
	}

	if viaFactory.BasePath != viaWrapper.RawService().BasePath {
		t.Errorf("expected equivalent base paths, got %q vs %q", viaFactory.BasePath, viaWrapper.RawService().BasePath)
	}
	if viaFactory.UserAgent != viaWrapper.RawService().UserAgent {
		t.Errorf("expected equivalent user agents, got %q vs %q", viaFactory.UserAgent, viaWrapper.RawService().UserAgent)
	}
}
