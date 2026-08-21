package drive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/api/drive/v3"
)

// Standard Google Drive Shared Drive permission roles.
const (
	RoleReader        = "reader"
	RoleCommenter     = "commenter"
	RoleWriter        = "writer"
	RoleFileOrganizer = "fileOrganizer"
	RoleOrganizer     = "organizer"
)

const (
	rankReader        = 10
	rankCommenter     = 20
	rankWriter        = 30
	rankFileOrganizer = 40
	rankOrganizer     = 50
)

var (
	// ErrEmptyMemberEmail indicates an empty member email address was provided.
	ErrEmptyMemberEmail = errors.New("drive: empty member email")

	// ErrUnknownRole indicates an unrecognized Google Drive role string.
	ErrUnknownRole = errors.New("drive: unknown role")

	// ErrNotFound indicates a requested Google Drive resource was not found.
	ErrNotFound = errors.New("drive: resource not found")
)

// RoleRank returns an integer severity rank for a Google Drive role.
// Higher ranks represent greater privileges. Returns 0 for unknown roles.
func RoleRank(role string) int {
	switch role {
	case RoleReader:
		return rankReader
	case RoleCommenter:
		return rankCommenter
	case RoleWriter:
		return rankWriter
	case RoleFileOrganizer:
		return rankFileOrganizer
	case RoleOrganizer:
		return rankOrganizer
	default:
		return 0
	}
}

// MembershipAction represents the reconciliation outcome of EnsureDriveMembership.
type MembershipAction string

const (
	// MembershipActionCreated indicates a new permission was provisioned.
	MembershipActionCreated MembershipAction = "CREATED"

	// MembershipActionUpgraded indicates an existing permission was upgraded to a higher role.
	MembershipActionUpgraded MembershipAction = "UPGRADED"

	// MembershipActionUnchanged indicates an existing permission already possessed the requested role or higher.
	MembershipActionUnchanged MembershipAction = "UNCHANGED"
)

// MembershipResult captures the outcome of an EnsureDriveMembership call.
type MembershipResult struct {
	DriveID      string           `json:"drive_id"`
	Member       string           `json:"member"`
	Role         string           `json:"role"`
	PreviousRole string           `json:"previous_role,omitempty"`
	Action       MembershipAction `json:"action"`
	DryRun       bool             `json:"dry_run"`
}

// EnsureDriveMembership idempotently ensures that the target member (email address) holds at least
// the specified role on the given Shared Drive.
//
// If the member is not present, the permission is created.
// If the member has a lower role, it is upgraded.
// If the member already has an equal or higher role, it is left unchanged.
// In dryRun mode, the result reflects the intended change without mutating the Drive.
func (s *Service) EnsureDriveMembership(ctx context.Context, driveID, memberEmail, role string, dryRun bool) (*MembershipResult, error) {
	memberEmail = strings.ToLower(strings.TrimSpace(memberEmail))
	if memberEmail == "" {
		return nil, fmt.Errorf("%w for drive %s", ErrEmptyMemberEmail, driveID)
	}

	targetRank := RoleRank(role)
	if targetRank == 0 {
		return nil, fmt.Errorf("%w %q (valid: %s, %s, %s, %s, %s)",
			ErrUnknownRole, role, RoleReader, RoleCommenter, RoleWriter, RoleFileOrganizer, RoleOrganizer)
	}

	perms, err := s.ListDriveMembers(ctx, driveID)
	if err != nil {
		return nil, fmt.Errorf("drive: list members for drive %s: %w", driveID, err)
	}

	res := &MembershipResult{
		DriveID: driveID,
		Member:  memberEmail,
		Role:    role,
		DryRun:  dryRun,
	}

	var current *drive.Permission
	for _, p := range perms {
		if strings.EqualFold(p.EmailAddress, memberEmail) {
			if current == nil || RoleRank(p.Role) > RoleRank(current.Role) {
				current = p
			}
		}
	}

	if current != nil {
		res.PreviousRole = current.Role
		if RoleRank(current.Role) >= targetRank {
			res.Action = MembershipActionUnchanged
			res.Role = current.Role

			return res, nil
		}

		res.Action = MembershipActionUpgraded
		if dryRun {
			return res, nil
		}

		s.log.Info("upgrading Shared Drive membership",
			slog.String("drive_id", driveID),
			slog.String("member", memberEmail),
			slog.String("from_role", current.Role),
			slog.String("to_role", role))

		_, err := s.drive.Permissions.Update(driveID, current.Id, &drive.Permission{Role: role}).
			SupportsAllDrives(true).
			UseDomainAdminAccess(true).
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("drive: upgrade member %s on drive %s to %s: %w", memberEmail, driveID, role, err)
		}

		return res, nil
	}

	res.Action = MembershipActionCreated
	if dryRun {
		return res, nil
	}

	s.log.Info("granting Shared Drive membership",
		slog.String("drive_id", driveID),
		slog.String("member", memberEmail),
		slog.String("role", role))

	newPerm := &drive.Permission{
		Type:         "user",
		Role:         role,
		EmailAddress: memberEmail,
	}

	_, err = s.drive.Permissions.Create(driveID, newPerm).
		SupportsAllDrives(true).
		UseDomainAdminAccess(true).
		SendNotificationEmail(false).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("drive: create member %s on drive %s (%s): %w", memberEmail, driveID, role, err)
	}

	return res, nil
}

// ListDriveMembers retrieves all permissions assigned directly to a Shared Drive.
func (s *Service) ListDriveMembers(ctx context.Context, driveID string) ([]*drive.Permission, error) {
	var out []*drive.Permission
	pageToken := ""

	for {
		call := s.drive.Permissions.List(driveID).
			SupportsAllDrives(true).
			UseDomainAdminAccess(true).
			PageSize(defaultAdminPageSize).
			Fields("nextPageToken,permissions(id,type,role,emailAddress,domain)").
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("drive: list permissions for drive %s: %w", driveID, err)
		}

		out = append(out, res.Permissions...)
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return out, nil
}

// ListFilePermissions lists all permissions attached to a specific file or folder.
func (s *Service) ListFilePermissions(ctx context.Context, fileID string) ([]*drive.Permission, error) {
	var out []*drive.Permission
	pageToken := ""

	for {
		call := s.drive.Permissions.List(fileID).
			SupportsAllDrives(true).
			PageSize(defaultAdminPageSize).
			Fields("nextPageToken,permissions(id,type,role,emailAddress,domain)").
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("drive: list permissions for file %s: %w", fileID, err)
		}

		out = append(out, res.Permissions...)
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return out, nil
}

// DeleteDriveMember deletes a permission from a Shared Drive using domain admin access.
func (s *Service) DeleteDriveMember(ctx context.Context, driveID, permissionID string) error {
	err := s.drive.Permissions.Delete(driveID, permissionID).
		SupportsAllDrives(true).
		UseDomainAdminAccess(true).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("drive: delete permission %s on drive %s: %w", permissionID, driveID, err)
	}

	return nil
}

// DeleteFilePermission removes a permission from a file or folder.
func (s *Service) DeleteFilePermission(ctx context.Context, fileID, permissionID string) error {
	err := s.drive.Permissions.Delete(fileID, permissionID).
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("drive: delete permission %s on file %s: %w", permissionID, fileID, err)
	}

	return nil
}
