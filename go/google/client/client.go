package client

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/resourcemanager"
	"github.com/duizendstra/alexandria/go/google/serviceusage"
	workspacedrive "github.com/duizendstra/alexandria/go/google/workspace/drive"
	admin "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/drive/v3"
	serviceusageapi "google.golang.org/api/serviceusage/v1"
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

// NewResourceManagerService creates a fully-authenticated Resource Manager Service using functional options.
func NewResourceManagerService(ctx context.Context, cfg resourcemanager.Config, opts ...auth.Option) (*resourcemanager.Service, error) {
	clientOpts, err := auth.ResolveClient(ctx, []string{cloudresourcemanager.CloudPlatformScope}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client: %w", err)
	}

	srv, err := resourcemanager.New(ctx, cfg, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create resourcemanager service: %w", err)
	}

	return srv, nil
}

// NewServiceUsageService creates a fully-authenticated Service Usage Service using functional options.
func NewServiceUsageService(ctx context.Context, cfg serviceusage.Config, opts ...auth.Option) (*serviceusage.Service, error) {
	clientOpts, err := auth.ResolveClient(ctx, []string{serviceusageapi.CloudPlatformScope}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client: %w", err)
	}

	srv, err := serviceusage.New(ctx, cfg, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create serviceusage service: %w", err)
	}

	return srv, nil
}
