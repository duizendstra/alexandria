package cloudrun

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// ErrSidecarContainersRequired means the service declares no containers.
	ErrSidecarContainersRequired = errors.New("cloudrun: at least one container is required")
	// ErrSidecarContainerNameRequired means a container has no name.
	ErrSidecarContainerNameRequired = errors.New("cloudrun: container Name is required")
	// ErrSidecarContainerImageRequired means a container has no image.
	ErrSidecarContainerImageRequired = errors.New("cloudrun: container Image is required")
	// ErrSidecarIngressPortRequired means not exactly one container declares Port.
	ErrSidecarIngressPortRequired = errors.New("cloudrun: exactly one container must declare Port")
)

// Container defines one container of a multi-container (sidecar) service.
type Container struct {
	// Name is the container name (a DNS_LABEL, unique within the service).
	Name string
	// Image is the initial container image.
	Image string
	// Port is the ingress container port. Exactly one container must
	// declare a port; the other containers are sidecars reachable over
	// localhost only.
	Port int
	// Memory is the memory limit for the container (e.g. "512Mi").
	Memory string
	// CPU is the CPU limit for the container (e.g. "1000m").
	// Cloud Run applies a server-side default when unset; declaring it
	// explicitly keeps the desired state aligned with the live resource.
	CPU string
	// Envs are the container's environment variables. Sidecars do not
	// receive PORT from Cloud Run — pass it explicitly when needed.
	Envs []EnvVar
	// DependsOn lists names of containers that must be started first.
	DependsOn []string
}

// SidecarServiceConfig defines a multi-container Cloud Run service:
// one ingress container plus sidecars sharing its network namespace.
type SidecarServiceConfig struct {
	// Name is the Cloud Run service name.
	Name string
	// Region is the GCP region.
	Region string
	// Containers in template order. Exactly one declares Port.
	Containers []Container
	// IAPEnabled fronts the service with Identity-Aware Proxy.
	IAPEnabled bool
	// MaxInstances caps scaling; 0 keeps the server-side default.
	MaxInstances int
}

// Validate checks that the sidecar service configuration is complete.
func (c *SidecarServiceConfig) Validate() error {
	if c.Name == "" {
		return ErrServiceNameRequired
	}
	if c.Region == "" {
		return ErrServiceRegionRequired
	}
	if len(c.Containers) == 0 {
		return ErrSidecarContainersRequired
	}

	ingress := 0
	for i := range c.Containers {
		if c.Containers[i].Name == "" {
			return ErrSidecarContainerNameRequired
		}
		if c.Containers[i].Image == "" {
			return ErrSidecarContainerImageRequired
		}
		if c.Containers[i].Port > 0 {
			ingress++
		}
	}
	if ingress != 1 {
		return ErrSidecarIngressPortRequired
	}

	return nil
}

// ApplySidecarService creates a multi-container Cloud Run v2 service.
// serviceAccountEmail may be nil to run as the project's default compute
// service account. Container image changes are ignored — deploys happen
// via CI/CD.
func ApplySidecarService(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg SidecarServiceConfig,
	serviceAccountEmail pulumi.StringInput,
	deps []pulumi.Resource,
) (*ServiceOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	containers := make(cloudrunv2.ServiceTemplateContainerArray, len(cfg.Containers))
	ignore := []string{ignoreClient, ignoreClientVersion}
	for i := range cfg.Containers {
		containers[i] = sidecarContainerArgs(&cfg.Containers[i])
		ignore = append(ignore, fmt.Sprintf("template.containers[%d].image", i))
	}

	tmpl := &cloudrunv2.ServiceTemplateArgs{Containers: containers}
	if serviceAccountEmail != nil {
		tmpl.ServiceAccount = serviceAccountEmail
	}
	if cfg.MaxInstances > 0 {
		tmpl.Scaling = &cloudrunv2.ServiceTemplateScalingArgs{
			MaxInstanceCount: pulumi.Int(cfg.MaxInstances),
		}
	}

	args := &cloudrunv2.ServiceArgs{
		Project:  projectID,
		Location: pulumi.String(cfg.Region),
		Name:     pulumi.String(cfg.Name),
		Template: tmpl,
	}
	if cfg.IAPEnabled {
		args.IapEnabled = pulumi.Bool(true)
	}

	svc, err := cloudrunv2.NewService(ctx, cfg.Name, args,
		pulumi.DependsOn(deps), pulumi.IgnoreChanges(ignore))
	if err != nil {
		return nil, fmt.Errorf("create cloud run sidecar service %s: %w", cfg.Name, err)
	}

	return &ServiceOutputs{
		Name: svc.Name,
		URI:  svc.Uri,
	}, nil
}

// sidecarContainerArgs maps one Container to its template args.
func sidecarContainerArgs(c *Container) *cloudrunv2.ServiceTemplateContainerArgs {
	envs := make(cloudrunv2.ServiceTemplateContainerEnvArray, len(c.Envs))
	for i, e := range c.Envs {
		envs[i] = &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String(e.Name),
			Value: e.Value,
		}
	}

	memLimit := c.Memory
	if memLimit == "" {
		memLimit = defaultMemory
	}
	limits := pulumi.StringMap{limitMemory: pulumi.String(memLimit)}
	if c.CPU != "" {
		limits[limitCPU] = pulumi.String(c.CPU)
	}

	a := &cloudrunv2.ServiceTemplateContainerArgs{
		Name:  pulumi.String(c.Name),
		Image: pulumi.String(c.Image),
		Envs:  envs,
		Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
			Limits: limits,
		},
	}
	if c.Port > 0 {
		a.Ports = &cloudrunv2.ServiceTemplateContainerPortsArgs{
			Name:          pulumi.String("http1"),
			ContainerPort: pulumi.Int(c.Port),
		}
	}
	if len(c.DependsOn) > 0 {
		a.DependsOns = pulumi.ToStringArray(c.DependsOn)
	}

	return a
}
