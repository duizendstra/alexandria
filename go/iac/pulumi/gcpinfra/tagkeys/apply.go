package tagkeys

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/tags"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ErrOrgIDRequired means no organization ID was provided.
// Test for it with [errors.Is].
var ErrOrgIDRequired = errors.New("tagkeys: orgID is required")

// Outputs maps dimension short name → Pulumi resource ID.
type Outputs map[string]pulumi.IDOutput

// Apply creates tag keys at the org level in GCP. Tag keys are protected from
// accidental deletion unless the caller passes lifecycle.Ephemeral.
func Apply(ctx *pulumi.Context, orgID string, dims []classification.Dimension, opts ...lifecycle.Option) (Outputs, error) {
	if orgID == "" {
		return nil, ErrOrgIDRequired
	}

	if err := classification.ValidateAll(dims); err != nil {
		return nil, fmt.Errorf("tagkeys: %w", err)
	}

	orgParent := pulumi.Sprintf("organizations/%s", orgID)
	outputs := Outputs{}

	for _, dim := range dims {
		tagKey, err := tags.NewTagKey(ctx, dim.ShortName, &tags.TagKeyArgs{
			Parent:      orgParent,
			ShortName:   pulumi.String(dim.ShortName),
			Description: pulumi.String(dim.Description),
		}, lifecycle.Protect(opts...))
		if err != nil {
			return nil, fmt.Errorf("create %s tag key: %w", dim.ShortName, err)
		}

		outputs[dim.ShortName] = tagKey.ID()
	}

	return outputs, nil
}
