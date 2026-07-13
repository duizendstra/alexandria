package drive

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// Default parameters for high-performance scanning to avoid magic number lints.
const (
	defaultPageSize = int64(1000)
	defaultMaxPages = 10000
)

// ErrNilService is returned when a nil Google Drive service client is provided to the scanner.
var ErrNilService = errors.New("google drive service must not be nil")

// Scanner orchestrates paginated flat file indexing operations against Google Drive.
type Scanner struct {
	pageSize int64
	maxPages int
	query    string
	fields   string
	corpora  string
	driveID  string
}

// ScannerOption defines a functional option configuration callback for our Scanner.
type ScannerOption func(*Scanner)

// WithPageSize configures the maximum number of file resources returned per API call.
func WithPageSize(size int64) ScannerOption {
	return func(s *Scanner) {
		if size > 0 {
			s.pageSize = size
		}
	}
}

// WithMaxPages sets a hard ceiling on the number of crawled pages to prevent execution runaway.
func WithMaxPages(limit int) ScannerOption {
	return func(s *Scanner) {
		s.maxPages = limit
	}
}

// WithQuery sets the Google Drive search filter (e.g. "mimeType = 'application/vnd.google-apps.folder'").
func WithQuery(query string) ScannerOption {
	return func(s *Scanner) {
		s.query = query
	}
}

// WithFields specifies which metadata fields to fetch (crucial for optimizing payload sizes).
func WithFields(fields string) ScannerOption {
	return func(s *Scanner) {
		if fields != "" {
			s.fields = fields
		}
	}
}

// WithCorpora configures the search corpus scope for the listing request (e.g. "allDrives").
func WithCorpora(corpora string) ScannerOption {
	return func(s *Scanner) {
		s.corpora = corpora
	}
}

// WithDriveID configures the specific Shared Drive ID scope to crawl (requires Corpora("drive")).
func WithDriveID(id string) ScannerOption {
	return func(s *Scanner) {
		s.driveID = id
	}
}

// NewScanner initializes a new Scanner instance with the provided options and sane defaults.
func NewScanner(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		pageSize: defaultPageSize,
		maxPages: defaultMaxPages,
		fields:   "nextPageToken, files(id, name, mimeType, parents, size, createdTime, driveId)",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Scan crawls Google Drive using flat list pagination and invokes the callback for each file resource.
func (s *Scanner) Scan(ctx context.Context, srv *drive.Service, onFile func(*drive.File) error) error {
	if srv == nil {
		return ErrNilService
	}

	var pageToken string
	pageCount := 0

	for {
		// Respect context cancellation gracefully.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Configure the API listing call.
		call := srv.Files.List().
			PageSize(s.pageSize).
			Fields(googleapi.Field(s.fields)).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			Context(ctx)

		if s.corpora != "" {
			call = call.Corpora(s.corpora)
		} else {
			call = call.Corpora("allDrives")
		}

		if s.driveID != "" {
			call = call.DriveId(s.driveID)
		}

		if s.query != "" {
			call = call.Q(s.query)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		// Execute the network request.
		res, err := call.Do()
		if err != nil {
			return fmt.Errorf("google drive API request failed at page %d: %w", pageCount+1, err)
		}

		// Stream the retrieved files back to the caller via callback function.
		for _, file := range res.Files {
			if file == nil {
				continue
			}
			if err := onFile(file); err != nil {
				return fmt.Errorf("processing callback failed for file %s: %w", file.Id, err)
			}
		}

		pageCount++
		pageToken = res.NextPageToken

		// Check if we've reached the terminal page or safety ceiling limit.
		if pageToken == "" {
			break
		}
		if s.maxPages > 0 && pageCount >= s.maxPages {
			break
		}
	}

	return nil
}
