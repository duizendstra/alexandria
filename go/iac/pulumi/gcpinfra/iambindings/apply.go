package iambindings

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Apply creates IAM member bindings from static config.
func Apply(ctx *pulumi.Context, bindings []Binding) error {
	for _, b := range bindings {
		if err := b.Validate(); err != nil {
			return err
		}

		_, err := projects.NewIAMMember(ctx, b.Name, &projects.IAMMemberArgs{
			Project: pulumi.String(b.ProjectID),
			Role:    pulumi.String(b.Role),
			Member:  pulumi.String(b.Member),
		})
		if err != nil {
			return fmt.Errorf("create IAM binding %s: %w", b.Name, err)
		}
	}

	return nil
}

// DynamicBinding grants a member a role on a project using Pulumi outputs.
// Use when project or member come from stack references.
type DynamicBinding struct {
	Name    string
	Project pulumi.StringInput
	Role    string
	Member  pulumi.StringInput
}

// ApplyDynamic creates IAM member bindings with dynamic project/member inputs.
func ApplyDynamic(ctx *pulumi.Context, bindings []DynamicBinding) ([]pulumi.Resource, error) {
	var resources []pulumi.Resource
	for _, b := range bindings {
		res, err := projects.NewIAMMember(ctx, b.Name, &projects.IAMMemberArgs{
			Project: b.Project,
			Role:    pulumi.String(b.Role),
			Member:  b.Member,
		})
		if err != nil {
			return nil, fmt.Errorf("create IAM binding %s: %w", b.Name, err)
		}
		resources = append(resources, res)
	}

	return resources, nil
}

// SAIamBinding grants a member a role on a service account.
// Use for SA-level IAM (e.g. iam.serviceAccountTokenCreator on a Dataform SA).
type SAIamBinding struct {
	Name             string
	ServiceAccountID pulumi.StringInput // projects/{project}/serviceAccounts/{email}.
	Role             string
	Member           pulumi.StringInput
}

// ApplySAIam creates IAM bindings on service accounts and returns the created resources.
func ApplySAIam(ctx *pulumi.Context, bindings []SAIamBinding) ([]pulumi.Resource, error) {
	var resources []pulumi.Resource
	for _, b := range bindings {
		res, err := serviceaccount.NewIAMMember(ctx, b.Name, &serviceaccount.IAMMemberArgs{
			ServiceAccountId: b.ServiceAccountID,
			Role:             pulumi.String(b.Role),
			Member:           b.Member,
		})
		if err != nil {
			return nil, fmt.Errorf("create SA IAM binding %s: %w", b.Name, err)
		}
		resources = append(resources, res)
	}

	return resources, nil
}
