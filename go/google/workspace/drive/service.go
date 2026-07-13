package drive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// File is a simplified view of a Google Drive file resource.
type File struct {
	ID       string
	Name     string
	MimeType string
	DriveID  string
}

// Config holds configuration parameters for our Service.
type Config struct {
	Logger *slog.Logger
}

// Service wraps the standard Google Drive API client.
type Service struct {
	drive *drive.Service
	log   *slog.Logger
}

// RawService returns the underlying raw Google Drive API client.
func (s *Service) RawService() *drive.Service {
	return s.drive
}

// New creates a new Service instance using the provided client options.
func New(ctx context.Context, cfg Config, clientOpts ...option.ClientOption) (*Service, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	svc, err := drive.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("drive: create service: %w", err)
	}

	return &Service{drive: svc, log: log}, nil
}

// FetchFile downloads a file's raw content bytes. Google Documents are exported
// to the target MIME type (defaulting to PDF); binary files are downloaded directly.
func (s *Service) FetchFile(ctx context.Context, id, sourceMimeType, targetMimeType string) ([]byte, error) {
	var (
		resp *http.Response
		err  error
	)

	if strings.HasPrefix(sourceMimeType, "application/vnd.google-apps.") {
		if targetMimeType == "" {
			targetMimeType = "application/pdf"
		}

		s.log.Info("exporting Google document",
			slog.String("id", id),
			slog.String("source", sourceMimeType),
			slog.String("target", targetMimeType))

		resp, err = s.drive.Files.Export(id, targetMimeType).Context(ctx).Download()
	} else {
		resp, err = s.drive.Files.Get(id).Context(ctx).Download()
	}

	if err != nil {
		return nil, fmt.Errorf("drive: fetch file: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("drive: read body: %w", err)
	}

	return body, nil
}

// Upload creates a new file resource on Google Drive with the provided media stream content.
func (s *Service) Upload(ctx context.Context, name, folderID string, properties map[string]string, content io.Reader) (*File, error) {
	driveFile := &drive.File{
		Name:       name,
		Properties: properties,
	}

	if folderID != "" {
		driveFile.Parents = []string{folderID}
	}

	created, err := s.drive.Files.Create(driveFile).
		Media(content).
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("drive: upload: %w", err)
	}

	return &File{
		ID:       created.Id,
		Name:     created.Name,
		MimeType: created.MimeType,
		DriveID:  created.DriveId,
	}, nil
}

type ServiceWrapper = Service
