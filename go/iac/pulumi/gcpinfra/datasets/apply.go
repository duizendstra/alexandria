package datasets

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/bigquery"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created dataset.
type Outputs struct {
	DatasetID pulumi.StringOutput
}

// Apply creates a BigQuery dataset in a GCP project.
//
// cfg.ID is the Pulumi logical name as well as the dataset ID: a stack that
// applies several datasets must give each a distinct ID, or the engine
// rejects the repeat as a duplicate URN. Changing it later replaces the
// dataset, so the dataset is protected unless the caller passes
// lifecycle.Ephemeral.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg Config, deps []pulumi.Resource, opts ...lifecycle.Option) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ephemeral := lifecycle.IsEphemeral(opts...)

	labels := make(pulumi.StringMap)
	for k, v := range cfg.Labels {
		labels[k] = pulumi.String(v)
	}

	ds, err := bigquery.NewDataset(ctx, cfg.ID, &bigquery.DatasetArgs{
		Project:      projectID,
		DatasetId:    pulumi.String(cfg.ID),
		FriendlyName: pulumi.String(cfg.FriendlyName),
		Description:  pulumi.String(cfg.Description),
		Location:     pulumi.String(cfg.Location),
		Labels:       labels,
		// Left false on a permanent stack: destroying a dataset that still
		// holds tables then fails at the provider, one layer under Protect.
		DeleteContentsOnDestroy: pulumi.Bool(ephemeral),
	}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
	if err != nil {
		return nil, fmt.Errorf("create dataset %s: %w", cfg.ID, err)
	}

	return &Outputs{DatasetID: ds.DatasetId}, nil
}
