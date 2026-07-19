package stackref_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/stackref"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/internals"
)

type mocks int

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	if args.TypeToken == "pulumi:pulumi:StackReference" {
		return args.Name, resource.PropertyMap{
			"outputs": resource.NewObjectProperty(resource.PropertyMap{
				"present": resource.NewStringProperty("value-1"),
			}),
		}, nil
	}

	return args.Name, args.Inputs, nil
}

func (mocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

func runRequireString(t *testing.T, key string) string {
	t.Helper()

	var got string

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		ref, err := pulumi.NewStackReference(ctx, "organization/example/stack", nil)
		if err != nil {
			return err
		}

		out, err := internals.UnsafeAwaitOutput(ctx.Context(), stackref.RequireString(ref, key))
		if err != nil {
			return err
		}

		got, _ = out.Value.(string)

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}

	return got
}

func TestRequireStringPresent(t *testing.T) {
	if v := runRequireString(t, "present"); v != "value-1" {
		t.Errorf("expected value-1, got %q", v)
	}
}

func TestRequireStringMissingIsEmpty(t *testing.T) {
	if v := runRequireString(t, "absent"); v != "" {
		t.Errorf("expected empty string, got %q", v)
	}
}
