package dataform_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/dataform"
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

func TestApplyCreatesRepoReleasesWorkflows(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := validRepo()
		repo.ForceDelete = true
		repo.Labels = map[string]string{"env": "test"}

		out, err := dataform.Apply(ctx, pulumi.String("proj").ToStringOutput(), &repo,
			pulumi.String("sa@example.iam.gserviceaccount.com"),
			[]dataform.ReleaseConfig{
				{Name: releaseProd, GitCommitish: branchMain, CronSchedule: cronDaily, TimeZone: "Europe/Amsterdam"},
			},
			[]dataform.WorkflowConfig{
				{
					Name: workflowNightly, ReleaseConfigName: releaseProd,
					CronSchedule: "0 7 * * *", TimeZone: "Europe/Amsterdam",
					IncludedTags: []string{"daily"}, IncludeTransitiveDeps: true,
				},
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

func TestApplyInvalidRepo(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := dataform.Apply(ctx, pulumi.String("proj").ToStringOutput(), &dataform.RepositoryConfig{},
			pulumi.String("sa@example.iam.gserviceaccount.com"), nil, nil, nil)
		if !errors.Is(err, dataform.ErrNameRequired) {
			t.Errorf("expected ErrNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestEnsureP4SA(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := dataform.EnsureP4SA(ctx, "dataform-p4sa", pulumi.String("proj").ToStringOutput())
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
