package logsinks

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/logging"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created log sink.
type Outputs struct {
	// WriterIdentity is the SA that writes logs — needs BQ writer role.
	WriterIdentity pulumi.StringOutput
}

// Apply creates an org-level log sink routing to a BigQuery destination.
func Apply(ctx *pulumi.Context, cfg Config, destination pulumi.StringOutput) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sink, err := logging.NewOrganizationSink(ctx, cfg.Name, &logging.OrganizationSinkArgs{
		OrgId:           pulumi.String(cfg.OrgID),
		Name:            pulumi.String(cfg.Name),
		Destination:     destination,
		Filter:          pulumi.String(cfg.Filter),
		IncludeChildren: pulumi.Bool(cfg.IncludeChildren),
	})
	if err != nil {
		return nil, fmt.Errorf("create log sink %s: %w", cfg.Name, err)
	}

	return &Outputs{WriterIdentity: sink.WriterIdentity}, nil
}
