package drive

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/api/drive/v3"
)

const (
	defaultAdminPageSize = 100
	searchPageSize       = 50
	queryPageSize        = 10
)

// SharedDriveOption configures optional parameters when creating a Shared Drive.
type SharedDriveOption func(*sharedDriveConfig)

type sharedDriveConfig struct {
	restrictions *drive.DriveRestrictions
}

// WithDriveRestrictions sets the access and sharing restrictions for the new Shared Drive.
func WithDriveRestrictions(r *drive.DriveRestrictions) SharedDriveOption {
	return func(cfg *sharedDriveConfig) {
		cfg.restrictions = r
	}
}

// CreateSharedDrive creates a new Google Shared Drive idempotently using requestID.
func (s *Service) CreateSharedDrive(ctx context.Context, name, requestID string, opts ...SharedDriveOption) (*drive.Drive, error) {
	var cfg sharedDriveConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	dr := &drive.Drive{
		Name:         name,
		Restrictions: cfg.restrictions,
	}

	call := s.drive.Drives.Create(requestID, dr).
		Context(ctx)

	s.log.Info("creating Shared Drive", slog.String("name", name), slog.String("request_id", requestID))

	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("drive: create shared drive %q (request_id=%s): %w", name, requestID, err)
	}

	return created, nil
}

// FindSharedDriveByName searches for a Shared Drive with an exact name match using domain admin access.
// If not found, returns ErrNotFound.
func (s *Service) FindSharedDriveByName(ctx context.Context, name string) (*drive.Drive, error) {
	escapedName := strings.ReplaceAll(name, "'", "\\'")
	q := fmt.Sprintf("name = '%s'", escapedName)

	call := s.drive.Drives.List().
		UseDomainAdminAccess(true).
		Q(q).
		Fields("drives(id,name,restrictions,hidden)").
		PageSize(searchPageSize).
		Context(ctx)

	list, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("drive: find shared drive by name %q: %w", name, err)
	}

	for _, d := range list.Drives {
		if d.Name == name {
			return d, nil
		}
	}

	return nil, fmt.Errorf("%w: shared drive %q", ErrNotFound, name)
}

// GetSharedDrive fetches a Shared Drive by ID using domain admin access.
func (s *Service) GetSharedDrive(ctx context.Context, driveID string) (*drive.Drive, error) {
	d, err := s.drive.Drives.Get(driveID).
		UseDomainAdminAccess(true).
		Fields("id,name,restrictions,hidden").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("drive: get shared drive %s: %w", driveID, err)
	}

	return d, nil
}

// ListSharedDrives lists all Shared Drives visible under domain admin access.
func (s *Service) ListSharedDrives(ctx context.Context, pageSize int64) ([]*drive.Drive, error) {
	if pageSize <= 0 {
		pageSize = defaultAdminPageSize
	}

	var out []*drive.Drive
	pageToken := ""

	for {
		call := s.drive.Drives.List().
			UseDomainAdminAccess(true).
			PageSize(pageSize).
			Fields("nextPageToken,drives(id,name,restrictions,hidden)").
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("drive: list shared drives: %w", err)
		}

		out = append(out, res.Drives...)
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return out, nil
}

// CreateFolder creates a new folder resource with optional appProperties.
func (s *Service) CreateFolder(ctx context.Context, parentID, name string, appProperties map[string]string) (*File, error) {
	driveFile := &drive.File{
		Name:          name,
		MimeType:      "application/vnd.google-apps.folder",
		AppProperties: appProperties,
	}

	if parentID != "" {
		driveFile.Parents = []string{parentID}
	}

	s.log.Info("creating Drive folder", slog.String("name", name), slog.String("parent_id", parentID))

	created, err := s.drive.Files.Create(driveFile).
		SupportsAllDrives(true).
		Fields("id,name,mimeType,driveId").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("drive: create folder %q: %w", name, err)
	}

	return &File{
		ID:       created.Id,
		Name:     created.Name,
		MimeType: created.MimeType,
		DriveID:  created.DriveId,
	}, nil
}

// FindFolder searches for a non-trashed folder by name inside a parent.
// Returns the folder ID or an empty string if absent.
func (s *Service) FindFolder(ctx context.Context, parentID, name string) (string, error) {
	escapedName := strings.ReplaceAll(name, "'", "\\'")
	q := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and '%s' in parents and trashed = false", escapedName, parentID)

	res, err := s.drive.Files.List().
		Q(q).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("files(id,name)").
		PageSize(queryPageSize).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("drive: find folder %q in %s: %w", name, parentID, err)
	}

	if len(res.Files) == 0 {
		return "", nil
	}

	return res.Files[0].Id, nil
}

// FindFolderByProperty searches for a non-trashed folder containing matching appProperties anywhere in accessible drives.
func (s *Service) FindFolderByProperty(ctx context.Context, key, value string) (string, error) {
	escKey := strings.ReplaceAll(key, "'", "\\'")
	escVal := strings.ReplaceAll(value, "'", "\\'")
	q := fmt.Sprintf("appProperties has { key='%s' and value='%s' } and mimeType = 'application/vnd.google-apps.folder' and trashed = false", escKey, escVal)

	res, err := s.drive.Files.List().
		Q(q).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("files(id,name)").
		PageSize(queryPageSize).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("drive: find folder by property %s=%s: %w", key, value, err)
	}

	if len(res.Files) == 0 {
		return "", nil
	}

	return res.Files[0].Id, nil
}

// MoveFile moves a file into newParentID while safely removing previous parent(s).
// If oldParentID is empty, it explicitly discovers existing parents to prevent orphaned multi-parent links or error swallowing.
func (s *Service) MoveFile(ctx context.Context, fileID, oldParentID, newParentID string) (*drive.File, error) {
	removeParents := oldParentID
	if removeParents == "" {
		existing, err := s.drive.Files.Get(fileID).
			SupportsAllDrives(true).
			Fields("id,parents").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("drive: move file %s: discover existing parents: %w", fileID, err)
		}

		removeParents = strings.Join(existing.Parents, ",")
	}

	call := s.drive.Files.Update(fileID, nil).
		AddParents(newParentID).
		SupportsAllDrives(true).
		Fields("id,name,parents,driveId")

	if removeParents != "" {
		call = call.RemoveParents(removeParents)
	}

	updated, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("drive: move file %s to %s (removing %s): %w", fileID, newParentID, removeParents, err)
	}

	return updated, nil
}

// SetAppProperties updates the custom application properties on a file or folder.
func (s *Service) SetAppProperties(ctx context.Context, fileID string, props map[string]string) error {
	_, err := s.drive.Files.Update(fileID, &drive.File{AppProperties: props}).
		SupportsAllDrives(true).
		Fields("id").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("drive: set app properties for %s: %w", fileID, err)
	}

	return nil
}

// TrashFile marks a file or folder as trashed.
func (s *Service) TrashFile(ctx context.Context, fileID string) (bool, error) {
	res, err := s.drive.Files.Update(fileID, &drive.File{Trashed: true}).
		SupportsAllDrives(true).
		Fields("id,trashed").
		Context(ctx).
		Do()
	if err != nil {
		return false, fmt.Errorf("drive: trash file %s: %w", fileID, err)
	}

	return res.Trashed, nil
}
