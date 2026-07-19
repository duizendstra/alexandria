package cloudrun

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// defaultMemory is the container memory limit when none is configured.
const defaultMemory = "512Mi"

// Resource-limit map keys and IgnoreChanges paths shared by the
// service, job, and sidecar-service appliers.
const (
	limitMemory         = "memory"
	limitCPU            = "cpu"
	ignoreClient        = "client"
	ignoreClientVersion = "clientVersion"
)

// ServiceOutputs holds references to the created Cloud Run service.
type ServiceOutputs struct {
	Name pulumi.StringOutput
	URI  pulumi.StringOutput
}

// ApplyService creates a Cloud Run v2 service.
// serviceAccountEmail and envs are separate params because they are dynamic
// Pulumi outputs (e.g. from NewAccount or other resource outputs).
// Container image changes are ignored — deploys happen via CI/CD.
func ApplyService(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg ServiceConfig, //nolint:gocritic // hugeParam: by-value keeps the v0.4.x API stable
	serviceAccountEmail pulumi.StringInput,
	envs []EnvVar,
	deps []pulumi.Resource,
) (*ServiceOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	svcEnvs := make(cloudrunv2.ServiceTemplateContainerEnvArray, len(envs))
	for i, e := range envs {
		svcEnvs[i] = &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String(e.Name),
			Value: e.Value,
		}
	}

	memLimit := cfg.Memory
	if memLimit == "" {
		memLimit = defaultMemory
	}

	svcLimits := pulumi.StringMap{limitMemory: pulumi.String(memLimit)}
	if cfg.CPU != "" {
		svcLimits[limitCPU] = pulumi.String(cfg.CPU)
	}

	svc, err := cloudrunv2.NewService(ctx, cfg.Name, &cloudrunv2.ServiceArgs{
		Project:  projectID,
		Location: pulumi.String(cfg.Region),
		Name:     pulumi.String(cfg.Name),
		Template: &cloudrunv2.ServiceTemplateArgs{
			ServiceAccount: serviceAccountEmail.ToStringOutput(),
			Containers: cloudrunv2.ServiceTemplateContainerArray{
				&cloudrunv2.ServiceTemplateContainerArgs{
					Image: pulumi.String(cfg.Image),
					Envs:  svcEnvs,
					Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
						Limits: svcLimits,
					},
				},
			},
		},
	}, pulumi.DependsOn(deps),
		pulumi.IgnoreChanges([]string{ignoreClient, ignoreClientVersion, "template.containers[0].image"}))
	if err != nil {
		return nil, fmt.Errorf("create cloud run service %s: %w", cfg.Name, err)
	}

	return &ServiceOutputs{
		Name: svc.Name,
		URI:  svc.Uri,
	}, nil
}

// JobOutputs holds references to the created Cloud Run job.
type JobOutputs struct {
	Name pulumi.StringOutput
}

// ApplyJob creates a Cloud Run v2 job.
// serviceAccountEmail and envs are separate params because they are dynamic.
// Container image changes are ignored — deploys happen via CI/CD.
func ApplyJob(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg JobConfig, //nolint:gocritic // hugeParam: by-value keeps the v0.4.x API stable
	serviceAccountEmail pulumi.StringInput,
	envs []EnvVar,
	deps []pulumi.Resource,
) (*JobOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	jobEnvs := make(cloudrunv2.JobTemplateTemplateContainerEnvArray, len(envs))
	for i, e := range envs {
		jobEnvs[i] = &cloudrunv2.JobTemplateTemplateContainerEnvArgs{
			Name:  pulumi.String(e.Name),
			Value: e.Value,
		}
	}

	jobMemLimit := cfg.Memory
	if jobMemLimit == "" {
		jobMemLimit = defaultMemory
	}

	jobLimits := pulumi.StringMap{limitMemory: pulumi.String(jobMemLimit)}
	if cfg.CPU != "" {
		jobLimits[limitCPU] = pulumi.String(cfg.CPU)
	}

	job, err := cloudrunv2.NewJob(ctx, cfg.Name, &cloudrunv2.JobArgs{
		Project:  projectID,
		Location: pulumi.String(cfg.Region),
		Name:     pulumi.String(cfg.Name),
		Template: &cloudrunv2.JobTemplateArgs{
			Template: &cloudrunv2.JobTemplateTemplateArgs{
				ServiceAccount: serviceAccountEmail.ToStringOutput(),
				Containers: cloudrunv2.JobTemplateTemplateContainerArray{
					&cloudrunv2.JobTemplateTemplateContainerArgs{
						Image: pulumi.String(cfg.Image),
						Envs:  jobEnvs,
						Resources: &cloudrunv2.JobTemplateTemplateContainerResourcesArgs{
							Limits: jobLimits,
						},
					},
				},
				MaxRetries: pulumi.Int(cfg.MaxRetries),
			},
		},
	}, pulumi.DependsOn(deps),
		pulumi.IgnoreChanges([]string{ignoreClient, ignoreClientVersion, "template.template.containers[0].image"}))
	if err != nil {
		return nil, fmt.Errorf("create cloud run job %s: %w", cfg.Name, err)
	}

	return &JobOutputs{Name: job.Name}, nil
}

// GrantInvoker grants the run.invoker role on a Cloud Run service.
func GrantInvoker(
	ctx *pulumi.Context,
	name string,
	projectID pulumi.StringOutput,
	location string,
	serviceName pulumi.StringOutput,
	member pulumi.StringInput,
) error {
	_, err := cloudrunv2.NewServiceIamMember(ctx, name, &cloudrunv2.ServiceIamMemberArgs{
		Project:  projectID,
		Location: pulumi.String(location),
		Name:     serviceName,
		Role:     pulumi.String("roles/run.invoker"),
		Member:   member,
	})
	if err != nil {
		return fmt.Errorf("grant invoker %s: %w", name, err)
	}

	return nil
}
