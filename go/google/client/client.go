package client

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/google/auth"
	workspacedrive "github.com/duizendstra/alexandria/go/google/workspace/drive"
	admin "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	"google.golang.org/api/drive/v3"
)

// NewDriveService creates a fully-authenticated Google Drive API client using functional options.
//
// It resolves authentication via auth.ResolveClient and delegates the actual
// service construction to workspace/drive.New, which is the single Drive
// service construction path in this module.
func NewDriveService(ctx context.Context, opts ...auth.Option) (*drive.Service, error) {
	clientOpts, err := auth.ResolveClient(ctx, []string{drive.DriveMetadataReadonlyScope}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client: %w", err)
	}

	srv, err := workspacedrive.New(ctx, workspacedrive.Config{}, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return srv.RawService(), nil
}

// NewAdminService creates a fully-authenticated Google Workspace Admin API client using functional options.
func NewAdminService(ctx context.Context, opts ...auth.Option) (*admin.Service, error) {
	clientOpts, err := auth.ResolveClient(ctx, []string{admin.AdminDirectoryUserReadonlyScope}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client: %w", err)
	}

	srv, err := admin.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin service: %w", err)
	}

	return srv, nil
}

// NewReportsService creates a fully-authenticated Google Workspace Admin Reports API client using functional options.
func NewReportsService(ctx context.Context, opts ...auth.Option) (*reports.Service, error) {
	clientOpts, err := auth.ResolveClient(ctx, []string{reports.AdminReportsAuditReadonlyScope}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client: %w", err)
	}

	srv, err := reports.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create reports service: %w", err)
	}

	return srv, nil
}
