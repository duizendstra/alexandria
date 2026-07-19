package scheduler_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/scheduler"
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

func TestApplyCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return scheduler.Apply(ctx, pulumi.String("proj").ToStringOutput(), &scheduler.Config{
			Name:     "heartbeat",
			Region:   "europe-west1",
			Schedule: "*/5 6-22 * * *",
			TimeZone: "Europe/Amsterdam",
			Paused:   true,
		}, pulumi.String("https://example.test/run"),
			pulumi.String("sa@example.iam.gserviceaccount.com"), nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := scheduler.Apply(ctx, pulumi.String("proj").ToStringOutput(), &scheduler.Config{},
			pulumi.String("https://example.test/run"),
			pulumi.String("sa@example.iam.gserviceaccount.com"), nil)
		if !errors.Is(err, scheduler.ErrNameRequired) {
			t.Errorf("expected ErrNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
