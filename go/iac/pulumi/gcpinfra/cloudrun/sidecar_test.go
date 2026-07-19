package cloudrun_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudrun"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func validSidecarConfig() cloudrun.SidecarServiceConfig {
	return cloudrun.SidecarServiceConfig{
		Name:   "frontend",
		Region: regionTest,
		Containers: []cloudrun.Container{
			{
				Name:   "webapp",
				Image:  "gcr.io/example/webapp:latest",
				Port:   3000,
				Memory: "256Mi",
				CPU:    testCPU,
				Envs:   []cloudrun.EnvVar{{Name: "NODE_ENV", Value: pulumi.String("production")}},
			},
			{
				Name:      "gatekeeper",
				Image:     "gcr.io/example/gatekeeper:latest",
				Memory:    "128Mi",
				CPU:       "500m",
				Envs:      []cloudrun.EnvVar{{Name: "PORT", Value: pulumi.String("8080")}},
				DependsOn: []string{"webapp"},
			},
		},
		IAPEnabled:   true,
		MaxInstances: 3,
	}
}

func TestApplySidecarServiceCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := cloudrun.ApplySidecarService(ctx, pulumi.String("proj").ToStringOutput(),
			validSidecarConfig(), nil, nil)
		if err != nil {
			return err
		}
		if out == nil {
			t.Error("expected outputs")
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplySidecarServiceWithServiceAccount(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := cloudrun.ApplySidecarService(ctx, pulumi.String("proj").ToStringOutput(),
			validSidecarConfig(), pulumi.String("sa@example.iam.gserviceaccount.com"), nil)
		if err != nil {
			return err
		}
		if out == nil {
			t.Error("expected outputs")
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplySidecarServiceInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*cloudrun.SidecarServiceConfig)
		wantErr error
	}{
		{
			name:    "missing name",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Name = "" },
			wantErr: cloudrun.ErrServiceNameRequired,
		},
		{
			name:    "missing region",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Region = "" },
			wantErr: cloudrun.ErrServiceRegionRequired,
		},
		{
			name:    "no containers",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Containers = nil },
			wantErr: cloudrun.ErrSidecarContainersRequired,
		},
		{
			name:    "missing container name",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Containers[0].Name = "" },
			wantErr: cloudrun.ErrSidecarContainerNameRequired,
		},
		{
			name:    "missing container image",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Containers[1].Image = "" },
			wantErr: cloudrun.ErrSidecarContainerImageRequired,
		},
		{
			name:    "no ingress port",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Containers[0].Port = 0 },
			wantErr: cloudrun.ErrSidecarIngressPortRequired,
		},
		{
			name:    "two ingress ports",
			mutate:  func(c *cloudrun.SidecarServiceConfig) { c.Containers[1].Port = 8080 },
			wantErr: cloudrun.ErrSidecarIngressPortRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cfg := validSidecarConfig()
				tt.mutate(&cfg)
				_, err := cloudrun.ApplySidecarService(ctx, pulumi.String("proj").ToStringOutput(),
					cfg, nil, nil)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}

				return nil
			}, pulumi.WithMocks("example", "stack", mocks(0)))
			if err != nil {
				t.Fatalf("pulumi run: %v", err)
			}
		})
	}
}
