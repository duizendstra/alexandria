package triggers

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudbuild"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Apply creates a Cloud Build trigger that fires on tag pushes.
func Apply(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	region string,
	repoID pulumi.IDOutput,
	triggerSA pulumi.StringOutput,
	cfg Config,
	deps []pulumi.Resource,
) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	subs := pulumi.StringMap{}
	for k, v := range cfg.Substitutions {
		subs[k] = pulumi.String(v)
	}

	args := &cloudbuild.TriggerArgs{
		Project:  projectID,
		Location: pulumi.String(region),
		Name:     pulumi.String(cfg.Name),
		RepositoryEventConfig: &cloudbuild.TriggerRepositoryEventConfigArgs{
			Repository: repoID,
			Push: &cloudbuild.TriggerRepositoryEventConfigPushArgs{
				Tag: pulumi.String(cfg.TagPattern),
			},
		},
		Filename:       pulumi.String(cfg.ConfigFile),
		ServiceAccount: triggerSA,
		Substitutions:  subs,
	}

	if cfg.RequireApproval {
		args.ApprovalConfig = &cloudbuild.TriggerApprovalConfigArgs{
			ApprovalRequired: pulumi.Bool(true),
		}
	}

	_, err := cloudbuild.NewTrigger(ctx, cfg.Name, args, pulumi.DependsOn(deps))
	if err != nil {
		return fmt.Errorf("create trigger %s: %w", cfg.Name, err)
	}

	return nil
}
