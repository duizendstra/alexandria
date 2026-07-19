package cloudrun_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudrun"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type mocks int

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	return args.Name + "_id", args.Inputs, nil
}

func (mocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

func TestApplyServiceCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := cloudrun.ApplyService(ctx, pulumi.String("proj").ToStringOutput(), cloudrun.ServiceConfig{
			Name:   "api",
			Region: regionTest,
			Image:  "gcr.io/example/api:latest",
			CPU:    testCPU,
		}, pulumi.String("sa@example.iam.gserviceaccount.com"), []cloudrun.EnvVar{
			{Name: "EXAMPLE", Value: pulumi.String("value")},
		}, nil)
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

func TestApplyServiceInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := cloudrun.ApplyService(ctx, pulumi.String("proj").ToStringOutput(), cloudrun.ServiceConfig{},
			pulumi.String("sa@example.iam.gserviceaccount.com"), nil, nil)
		if !errors.Is(err, cloudrun.ErrServiceNameRequired) {
			t.Errorf("expected ErrServiceNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyJobCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := cloudrun.ApplyJob(ctx, pulumi.String("proj").ToStringOutput(), cloudrun.JobConfig{
			Name:       "worker",
			Region:     regionTest,
			Image:      "gcr.io/example/worker:latest",
			MaxRetries: 1,
			Memory:     "1Gi",
			CPU:        testCPU,
		}, pulumi.String("sa@example.iam.gserviceaccount.com"), []cloudrun.EnvVar{
			{Name: "EXAMPLE", Value: pulumi.String("value")},
		}, nil)
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

func TestApplyJobInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := cloudrun.ApplyJob(ctx, pulumi.String("proj").ToStringOutput(), cloudrun.JobConfig{},
			pulumi.String("sa@example.iam.gserviceaccount.com"), nil, nil)
		if !errors.Is(err, cloudrun.ErrJobNameRequired) {
			t.Errorf("expected ErrJobNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestGrantInvoker(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return cloudrun.GrantInvoker(ctx, "worker-invokes-api",
			pulumi.String("proj").ToStringOutput(), regionTest,
			pulumi.String("api").ToStringOutput(),
			pulumi.String("serviceAccount:sa@example.iam.gserviceaccount.com"))
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
