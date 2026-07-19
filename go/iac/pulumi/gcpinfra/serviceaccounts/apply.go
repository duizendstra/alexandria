package serviceaccounts

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs maps account ID → email output.
type Outputs map[string]pulumi.StringOutput

// Apply creates service accounts in a GCP project.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, accounts []Account, deps []pulumi.Resource) (Outputs, error) {
	outputs := Outputs{}

	for _, a := range accounts {
		if err := a.Validate(); err != nil {
			return nil, err
		}

		sa, err := serviceaccount.NewAccount(ctx, a.ID, &serviceaccount.AccountArgs{
			Project:     projectID,
			AccountId:   pulumi.String(a.ID),
			DisplayName: pulumi.String(a.DisplayName),
		}, pulumi.DependsOn(deps))
		if err != nil {
			return nil, fmt.Errorf("create SA %s: %w", a.ID, err)
		}

		outputs[a.ID] = sa.Email
	}

	return outputs, nil
}
