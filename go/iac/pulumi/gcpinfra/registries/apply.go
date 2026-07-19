package registries

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created registry.
type Outputs struct {
	RepositoryID pulumi.IDOutput
	Name         pulumi.StringOutput
}

// Apply creates an Artifact Registry repository in a GCP project.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg Config, deps []pulumi.Resource) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	repo, err := artifactregistry.NewRepository(ctx, cfg.ID, &artifactregistry.RepositoryArgs{
		Project:      projectID,
		Location:     pulumi.String(cfg.Location),
		RepositoryId: pulumi.String(cfg.ID),
		Format:       pulumi.String(cfg.Format),
		Description:  pulumi.String(cfg.Description),
	}, pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("create registry %s: %w", cfg.ID, err)
	}

	return &Outputs{
		RepositoryID: repo.ID(),
		Name:         repo.Name,
	}, nil
}

// GrantReader grants a member read access to the repository.
// Used to let consumer project Cloud Run agents pull images.
func GrantReader(
	ctx *pulumi.Context,
	name string,
	projectID pulumi.StringOutput,
	location string,
	repoName pulumi.StringOutput,
	member pulumi.StringInput,
) error {
	_, err := artifactregistry.NewRepositoryIamMember(ctx, name, &artifactregistry.RepositoryIamMemberArgs{
		Project:    projectID,
		Location:   pulumi.String(location),
		Repository: repoName,
		Role:       pulumi.String("roles/artifactregistry.reader"),
		Member:     member,
	})
	if err != nil {
		return fmt.Errorf("grant AR reader %s: %w", name, err)
	}

	return nil
}

// GrantWriter grants a member write access to the repository.
// Used for build service accounts that push images.
func GrantWriter(ctx *pulumi.Context, name string, projectID pulumi.StringOutput, location string, repoName, member pulumi.StringOutput) error {
	_, err := artifactregistry.NewRepositoryIamMember(ctx, name, &artifactregistry.RepositoryIamMemberArgs{
		Project:    projectID,
		Location:   pulumi.String(location),
		Repository: repoName,
		Role:       pulumi.String("roles/artifactregistry.writer"),
		Member:     member,
	})
	if err != nil {
		return fmt.Errorf("grant AR writer %s: %w", name, err)
	}

	return nil
}
