package projects

import (
	"fmt"

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
// The project is protected from accidental deletion and has no default VPC.
func Apply(ctx *pulumi.Context, cfg Config) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	project, err := organizations.NewProject(ctx, cfg.Name, &organizations.ProjectArgs{
		Name:              pulumi.String(cfg.Name),
		ProjectId:         pulumi.String(cfg.Name),
		FolderId:          pulumi.String(cfg.FolderID),
		BillingAccount:    pulumi.String(cfg.BillingAccount),
		DeletionPolicy:    pulumi.String("PREVENT"),
		AutoCreateNetwork: pulumi.Bool(false),
	})
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
