package uptimechecks_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/uptimechecks"
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
		// IAP-protected endpoint: accept the sign-in redirect (3xx) too.
		return uptimechecks.Apply(ctx, pulumi.String("proj").ToStringOutput(), &uptimechecks.Config{
			DisplayName:           "portal prod",
			Host:                  "portal.example.com",
			Path:                  "/",
			AcceptedStatusClasses: []string{uptimechecks.Class2xx, uptimechecks.Class3xx},
		}, pulumi.StringArray{pulumi.String("channel_id")}, nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyCreatesWithDefaults(t *testing.T) {
	// Only the required fields set — exercises the default-filling path.
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return uptimechecks.Apply(ctx, pulumi.String("proj").ToStringOutput(), &uptimechecks.Config{
			DisplayName: "minimal",
			Host:        "example.com",
		}, pulumi.StringArray{pulumi.String("channel_id")}, nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := uptimechecks.Apply(ctx, pulumi.String("proj").ToStringOutput(), &uptimechecks.Config{},
			pulumi.StringArray{}, nil)
		if !errors.Is(err, uptimechecks.ErrDisplayNameRequired) {
			t.Errorf("expected ErrDisplayNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
