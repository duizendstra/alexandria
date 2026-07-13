package drive_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/google/workspace/drive"

	google_drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// mockRoundTripper intercepts outgoing HTTP requests and delegates them to a custom function.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestScanner_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		opts          []drive.ScannerOption
		roundTripFunc func(req *http.Request) (*http.Response, error)
		expectedFiles []string
		expectedPages int
		expectError   bool
	}{
		{
			name: "Happy Path - Single Page",
			opts: nil, // Use defaults.
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				jsonResponse := `{
					"files": [
						{"id": "file-1", "name": "Document 1", "mimeType": "application/pdf"},
						{"id": "file-2", "name": "Document 2", "mimeType": "image/png"}
					]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
					Header:     make(http.Header),
				}, nil
			},
			expectedFiles: []string{"file-1", "file-2"},
			expectedPages: 1,
			expectError:   false,
		},
		{
			name: "Happy Path - Paginated Listing",
			opts: []drive.ScannerOption{
				drive.WithPageSize(1),
				drive.WithMaxPages(5),
			},
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				var jsonResponse string
				q := req.URL.Query()

				switch pageToken := q.Get("pageToken"); pageToken {
				case "":
					jsonResponse = `{
						"nextPageToken": "token-page-2",
						"files": [{"id": "file-page-1", "name": "Doc Page 1"}]
					}`
				case "token-page-2":
					jsonResponse = `{
						"files": [{"id": "file-page-2", "name": "Doc Page 2"}]
					}`
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
					Header:     make(http.Header),
				}, nil
			},
			expectedFiles: []string{"file-page-1", "file-page-2"},
			expectedPages: 2,
			expectError:   false,
		},
		{
			name: "Error - API Failure",
			opts: nil,
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": "Internal Server Error"}`)),
					Header:     make(http.Header),
				}, nil
			},
			expectedFiles: nil,
			expectedPages: 0,
			expectError:   true,
		},
		{
			name: "Limit - Max Pages Hit",
			opts: []drive.ScannerOption{
				drive.WithPageSize(1),
				drive.WithMaxPages(1),
			},
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				jsonResponse := `{
					"nextPageToken": "token-page-2",
					"files": [{"id": "file-page-1", "name": "Doc Page 1"}]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
					Header:     make(http.Header),
				}, nil
			},
			expectedFiles: []string{"file-page-1"},
			expectedPages: 1,
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Initialize Google Drive service using custom Mock RoundTripper.
			mockClient := &http.Client{
				Transport: &mockRoundTripper{roundTripFunc: tc.roundTripFunc},
			}
			srv, err := google_drive.NewService(context.Background(), option.WithHTTPClient(mockClient))
			if err != nil {
				t.Fatalf("failed to initialize drive mock service: %v", err)
			}

			// Instantiate our scanner and collect callback outputs.
			scanner := drive.NewScanner(tc.opts...)
			var capturedFiles []string

			err = scanner.Scan(context.Background(), srv, func(file *google_drive.File) error {
				capturedFiles = append(capturedFiles, file.Id)
				return nil
			})

			// Evaluate results.
			if tc.expectError {
				if err == nil {
					t.Errorf("expected Scan error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("expected no Scan error, but got: %v", err)
			}

			if len(capturedFiles) != len(tc.expectedFiles) {
				t.Errorf("expected %d files, got %d files (%v)", len(tc.expectedFiles), len(capturedFiles), capturedFiles)
			}

			for i, expectedID := range tc.expectedFiles {
				if i < len(capturedFiles) && capturedFiles[i] != expectedID {
					t.Errorf("at index %d: expected file ID %q, got %q", i, expectedID, capturedFiles[i])
				}
			}
		})
	}
}

func TestScanner_Scan_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Arrange context with early cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Trigger instant cancellation.

	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				t.Error("roundtripper should not be called when context is pre-cancelled")
				return nil, errors.New("aborted")
			},
		},
	}
	srv, err := google_drive.NewService(context.Background(), option.WithHTTPClient(mockClient))
	if err != nil {
		t.Fatalf("failed to initialize drive mock service: %v", err)
	}

	scanner := drive.NewScanner()
	err = scanner.Scan(ctx, srv, func(file *google_drive.File) error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestScanner_Scan_CallbackError(t *testing.T) {
	t.Parallel()

	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				jsonResponse := `{"files": [{"id": "file-1"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	srv, err := google_drive.NewService(context.Background(), option.WithHTTPClient(mockClient))
	if err != nil {
		t.Fatalf("failed to initialize drive mock service: %v", err)
	}

	scanner := drive.NewScanner()
	customErr := errors.New("custom error")

	err = scanner.Scan(context.Background(), srv, func(file *google_drive.File) error {
		return customErr
	})

	if err == nil || !strings.Contains(err.Error(), "processing callback failed") {
		t.Errorf("expected error wrapping processing callback failure, got: %v", err)
	}
}
