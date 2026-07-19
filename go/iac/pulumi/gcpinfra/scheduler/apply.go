package scheduler

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudscheduler"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// defaultHTTPMethod is used when no HTTPMethod is configured.
const defaultHTTPMethod = "POST"

// Apply creates a Cloud Scheduler job with HTTP target and OAuth authentication.
// targetURI and oauthSAEmail are separate params because they are dynamic
// Pulumi outputs (constructed from other resource outputs).
func Apply(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg *Config,
	targetURI pulumi.StringInput,
	oauthSAEmail pulumi.StringInput,
	deps []pulumi.Resource,
) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	httpMethod := cfg.HTTPMethod
	if httpMethod == "" {
		httpMethod = defaultHTTPMethod
	}

	_, err := cloudscheduler.NewJob(ctx, cfg.Name, &cloudscheduler.JobArgs{
		Project:  projectID,
		Region:   pulumi.String(cfg.Region),
		Name:     pulumi.String(cfg.Name),
		Schedule: pulumi.String(cfg.Schedule),
		TimeZone: pulumi.String(cfg.TimeZone),
		Paused:   pulumi.Bool(cfg.Paused),
		HttpTarget: &cloudscheduler.JobHttpTargetArgs{
			Uri:        targetURI.ToStringOutput(),
			HttpMethod: pulumi.String(httpMethod),
			OauthToken: &cloudscheduler.JobHttpTargetOauthTokenArgs{
				ServiceAccountEmail: oauthSAEmail.ToStringOutput(),
			},
		},
	}, pulumi.DependsOn(deps))
	if err != nil {
		return fmt.Errorf("create scheduler job %s: %w", cfg.Name, err)
	}

	return nil
}
