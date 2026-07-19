package dataform

import (
	"fmt"

	gcpdataform "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dataform"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created Dataform resources.
type Outputs struct {
	RepositoryName pulumi.StringOutput
}

// Apply creates a Dataform repository with optional release and workflow configs.
// serviceAccountEmail is passed separately because it is a dynamic Pulumi output
// (e.g. from a stack reference or NewAccount).
func Apply(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	repo *RepositoryConfig,
	serviceAccountEmail pulumi.StringInput,
	releases []ReleaseConfig,
	workflows []WorkflowConfig,
	deps []pulumi.Resource,
) (*Outputs, error) {
	if err := repo.Validate(); err != nil {
		return nil, err
	}

	labels := make(pulumi.StringMap)
	for k, v := range repo.Labels {
		labels[k] = pulumi.String(v)
	}

	dfRepoArgs := &gcpdataform.RepositoryArgs{
		Project:        projectID,
		Region:         pulumi.String(repo.Region),
		Name:           pulumi.String(repo.Name),
		DisplayName:    pulumi.String(repo.DisplayName),
		ServiceAccount: serviceAccountEmail.ToStringOutput(),
		Labels:         labels,
		GitRemoteSettings: &gcpdataform.RepositoryGitRemoteSettingsArgs{
			Url:                              pulumi.String(repo.GitURL),
			DefaultBranch:                    pulumi.String(repo.DefaultBranch),
			AuthenticationTokenSecretVersion: pulumi.String(repo.TokenSecretVersion),
		},
	}

	if repo.ForceDelete {
		dfRepoArgs.DeletionPolicy = pulumi.String("FORCE")
	}

	dfRepo, err := gcpdataform.NewRepository(ctx, repo.Name, dfRepoArgs, pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("create dataform repository %s: %w", repo.Name, err)
	}

	if err := applyReleases(ctx, projectID, repo, dfRepo, releases); err != nil {
		return nil, err
	}

	if err := applyWorkflows(ctx, projectID, repo, dfRepo, serviceAccountEmail, workflows); err != nil {
		return nil, err
	}

	return &Outputs{RepositoryName: dfRepo.Name}, nil
}

// applyReleases creates the release configs on the repository.
func applyReleases(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	repo *RepositoryConfig,
	dfRepo *gcpdataform.Repository,
	releases []ReleaseConfig,
) error {
	for _, rc := range releases {
		if err := rc.Validate(); err != nil {
			return err
		}

		rcArgs := &gcpdataform.RepositoryReleaseConfigArgs{
			Project:      projectID,
			Region:       pulumi.String(repo.Region),
			Repository:   dfRepo.Name,
			Name:         pulumi.String(rc.Name),
			GitCommitish: pulumi.String(rc.GitCommitish),
			CodeCompilationConfig: &gcpdataform.RepositoryReleaseConfigCodeCompilationConfigArgs{
				DefaultDatabase: rc.DefaultDatabase,
				Vars:            rc.Vars,
			},
		}

		if rc.CronSchedule != "" {
			rcArgs.CronSchedule = pulumi.String(rc.CronSchedule)
			rcArgs.TimeZone = pulumi.String(rc.TimeZone)
		}

		_, err := gcpdataform.NewRepositoryReleaseConfig(ctx, repo.Name+"-release-"+rc.Name, rcArgs,
			pulumi.DependsOn([]pulumi.Resource{dfRepo}))
		if err != nil {
			return fmt.Errorf("create release config %s: %w", rc.Name, err)
		}
	}

	return nil
}

// applyWorkflows creates the workflow configs on the repository.
func applyWorkflows(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	repo *RepositoryConfig,
	dfRepo *gcpdataform.Repository,
	serviceAccountEmail pulumi.StringInput,
	workflows []WorkflowConfig,
) error {
	for _, wc := range workflows {
		if err := wc.Validate(); err != nil {
			return err
		}

		releaseConfigPath := pulumi.Sprintf(
			"projects/%s/locations/%s/repositories/%s/releaseConfigs/%s",
			projectID, repo.Region, dfRepo.Name, wc.ReleaseConfigName,
		)

		invocationConfig := &gcpdataform.RepositoryWorkflowConfigInvocationConfigArgs{
			ServiceAccount: serviceAccountEmail.ToStringOutput(),
		}

		if len(wc.IncludedTags) > 0 {
			tags := make(pulumi.StringArray, len(wc.IncludedTags))
			for i, t := range wc.IncludedTags {
				tags[i] = pulumi.String(t)
			}

			invocationConfig.IncludedTags = tags
			if wc.IncludeTransitiveDeps {
				invocationConfig.TransitiveDependenciesIncluded = pulumi.Bool(true)
			}
		}

		wfArgs := &gcpdataform.RepositoryWorkflowConfigArgs{
			Project:          projectID,
			Region:           pulumi.String(repo.Region),
			Repository:       dfRepo.Name,
			Name:             pulumi.String(wc.Name),
			ReleaseConfig:    releaseConfigPath,
			InvocationConfig: invocationConfig,
		}

		if wc.CronSchedule != "" {
			wfArgs.CronSchedule = pulumi.String(wc.CronSchedule)
			wfArgs.TimeZone = pulumi.String(wc.TimeZone)
		}

		_, err := gcpdataform.NewRepositoryWorkflowConfig(ctx, repo.Name+"-workflow-"+wc.Name, wfArgs,
			pulumi.DependsOn([]pulumi.Resource{dfRepo}))
		if err != nil {
			return fmt.Errorf("create workflow config %s: %w", wc.Name, err)
		}
	}

	return nil
}

// P4SAOutputs holds the Dataform per-product service agent outputs.
type P4SAOutputs struct {
	Email pulumi.StringOutput
	P4SA  *projects.ServiceIdentity
}

// EnsureP4SA creates the Dataform per-product service agent on a project.
// Returns the P4SA email for IAM binding.
func EnsureP4SA(
	ctx *pulumi.Context,
	name string,
	projectID pulumi.StringOutput,
) (*P4SAOutputs, error) {
	p4sa, err := projects.NewServiceIdentity(ctx, name, &projects.ServiceIdentityArgs{
		Project: projectID,
		Service: pulumi.String("dataform.googleapis.com"),
	})
	if err != nil {
		return nil, fmt.Errorf("create dataform P4SA: %w", err)
	}

	return &P4SAOutputs{
		Email: p4sa.Email,
		P4SA:  p4sa,
	}, nil
}
