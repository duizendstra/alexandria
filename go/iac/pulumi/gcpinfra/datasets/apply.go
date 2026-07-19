package datasets

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/bigquery"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created dataset.
type Outputs struct {
	DatasetID pulumi.StringOutput
}

// Apply creates a BigQuery dataset in a GCP project.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg Config, deps []pulumi.Resource) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

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
	}, pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("create dataset %s: %w", cfg.ID, err)
	}

	return &Outputs{DatasetID: ds.DatasetId}, nil
}
