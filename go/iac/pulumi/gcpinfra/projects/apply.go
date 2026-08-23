package projects

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	gcpprojects "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created project.
type Outputs struct {
	// ProjectID is the project identifier string.
	ProjectID pulumi.StringOutput
	// ProjectNumber is the numeric project identifier.
	ProjectNumber pulumi.StringOutput
}

// Apply creates a GCP project with API enablement.
//
// The project has no default VPC and is protected from accidental deletion at
// both layers unless the caller passes lifecycle.Ephemeral. DeletionPolicy
// alone would fail the rename mid-apply; Protect refuses it at preview.
func Apply(ctx *pulumi.Context, cfg Config, opts ...lifecycle.Option) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	deletionPolicy := "PREVENT"
	if lifecycle.IsEphemeral(opts...) {
		deletionPolicy = "DELETE"
	}

	project, err := organizations.NewProject(ctx, cfg.Name, &organizations.ProjectArgs{
		Name:              pulumi.String(cfg.Name),
		ProjectId:         pulumi.String(cfg.Name),
		FolderId:          pulumi.String(cfg.FolderID),
		BillingAccount:    pulumi.String(cfg.BillingAccount),
		DeletionPolicy:    pulumi.String(deletionPolicy),
		AutoCreateNetwork: pulumi.Bool(false),
	}, lifecycle.Protect(opts...))
	if err != nil {
		return nil, fmt.Errorf("create project %s: %w", cfg.Name, err)
	}

	// Enable APIs sequentially — each depends on the project.
	for _, api := range cfg.APIs {
		_, err = gcpprojects.NewService(ctx, cfg.Name+"-"+api, &gcpprojects.ServiceArgs{
			Project:                  project.ProjectId,
			Service:                  pulumi.String(api),
			DisableDependentServices: pulumi.Bool(false),
			DisableOnDestroy:         pulumi.Bool(false),
		}, pulumi.DependsOn([]pulumi.Resource{project}))
		if err != nil {
			return nil, fmt.Errorf("enable API %s on %s: %w", api, cfg.Name, err)
		}
	}

	return &Outputs{
		ProjectID:     project.ProjectId,
		ProjectNumber: project.Number,
	}, nil
}
