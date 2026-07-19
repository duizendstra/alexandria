package folders

import (
	"errors"
	"fmt"
	"strings"

	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/governance/scope"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ErrInvalidParent means the parent is not a GCP organizations/ or
// folders/ resource path. Test for it with [errors.Is].
var ErrInvalidParent = errors.New(`folders: parent must start with "organizations/" or "folders/"`)

// Outputs holds the created folder references.
type Outputs struct {
	// RootFolderID is the numeric ID of the root folder.
	RootFolderID pulumi.StringOutput
	// FolderIDs maps child name → folder numeric ID.
	FolderIDs pulumi.Map
}

// ParseScope determines governance scope from a GCP parent resource path.
// GCP parents look like "organizations/123456" or "folders/789012".
// This is GCP-specific validation — the domain only knows about Scope.
func ParseScope(parent string) (scope.Scope, error) {
	switch {
	case strings.HasPrefix(parent, "organizations/"):
		return scope.Organization, nil
	case strings.HasPrefix(parent, "folders/"):
		return scope.Container, nil
	default:
		return 0, fmt.Errorf("%w, got %q", ErrInvalidParent, parent)
	}
}

// OrgID extracts the numeric org ID from a GCP parent string.
// Non-org parents pass through unchanged — callers gate on scope first.
func OrgID(parent string) string {
	return strings.TrimPrefix(parent, "organizations/")
}

// Apply creates the folder hierarchy in GCP. Idempotent via Pulumi state.
// Protected from accidental deletion at both GCP and Pulumi level.
//
// Only well-formedness is validated here (parent, root name, child
// uniqueness). Whether children are required at all is tier policy owned
// by the plan package — a Starter hierarchy legitimately has none.
func Apply(ctx *pulumi.Context, cfg hierarchy.Config) (*Outputs, error) {
	if err := cfg.ValidateBase(); err != nil {
		return nil, fmt.Errorf("folders: %w", err)
	}

	if err := cfg.ValidateChildren(); err != nil {
		return nil, fmt.Errorf("folders: %w", err)
	}

	// GCP-specific: validate parent format.
	if _, err := ParseScope(cfg.Parent); err != nil {
		return nil, err
	}

	rootFolder, err := organizations.NewFolder(ctx, cfg.RootName, &organizations.FolderArgs{
		Parent:             pulumi.String(cfg.Parent),
		DisplayName:        pulumi.String(cfg.RootName),
		DeletionProtection: pulumi.Bool(true),
	}, pulumi.Protect(true))
	if err != nil {
		return nil, fmt.Errorf("create %s folder: %w", cfg.RootName, err)
	}

	folderIDs := pulumi.Map{}

	for _, child := range cfg.Children {
		folder, err := organizations.NewFolder(ctx, child, &organizations.FolderArgs{
			Parent:             rootFolder.Name,
			DisplayName:        pulumi.String(child),
			DeletionProtection: pulumi.Bool(true),
		}, pulumi.Protect(true))
		if err != nil {
			return nil, fmt.Errorf("create %s folder: %w", child, err)
		}

		folderIDs[child] = folder.FolderId
	}

	return &Outputs{
		RootFolderID: rootFolder.FolderId,
		FolderIDs:    folderIDs,
	}, nil
}
