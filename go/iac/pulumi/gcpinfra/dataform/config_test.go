package dataform_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/dataform"
)

const (
	branchMain      = "main"
	releaseProd     = "prod"
	workflowNightly = "nightly"
	cronDaily       = "0 6 * * *"
)

func validRepo() dataform.RepositoryConfig {
	return dataform.RepositoryConfig{
		Name:               "warehouse",
		Region:             "europe-west4",
		GitURL:             "https://github.com/example/warehouse.git",
		DefaultBranch:      branchMain,
		TokenSecretVersion: "projects/example/secrets/git-token/versions/latest",
	}
}

func TestRepositoryValidateValid(t *testing.T) {
	c := validRepo()
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepositoryValidateMissingFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*dataform.RepositoryConfig)
		want   error
	}{
		{"name", func(c *dataform.RepositoryConfig) { c.Name = "" }, dataform.ErrNameRequired},
		{"region", func(c *dataform.RepositoryConfig) { c.Region = "" }, dataform.ErrRegionRequired},
		{"gitURL", func(c *dataform.RepositoryConfig) { c.GitURL = "" }, dataform.ErrGitURLRequired},
		{"defaultBranch", func(c *dataform.RepositoryConfig) { c.DefaultBranch = "" }, dataform.ErrDefaultBranchRequired},
		{"tokenSecretVersion", func(c *dataform.RepositoryConfig) { c.TokenSecretVersion = "" }, dataform.ErrTokenSecretVersionRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validRepo()
			tc.mutate(&c)
			if err := c.Validate(); !errors.Is(err, tc.want) {
				t.Errorf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestReleaseValidateValid(t *testing.T) {
	c := dataform.ReleaseConfig{Name: releaseProd, GitCommitish: branchMain}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReleaseValidateMissingName(t *testing.T) {
	c := dataform.ReleaseConfig{GitCommitish: branchMain}
	if err := c.Validate(); !errors.Is(err, dataform.ErrReleaseNameRequired) {
		t.Errorf("expected ErrReleaseNameRequired, got %v", err)
	}
}

func TestReleaseValidateMissingGitCommitish(t *testing.T) {
	c := dataform.ReleaseConfig{Name: releaseProd}
	if err := c.Validate(); !errors.Is(err, dataform.ErrReleaseGitCommitishRequired) {
		t.Errorf("expected ErrReleaseGitCommitishRequired, got %v", err)
	}
}

func TestReleaseValidateScheduleWithoutTimeZone(t *testing.T) {
	c := dataform.ReleaseConfig{Name: releaseProd, GitCommitish: branchMain, CronSchedule: cronDaily}
	if err := c.Validate(); !errors.Is(err, dataform.ErrReleaseTimeZoneRequired) {
		t.Errorf("expected ErrReleaseTimeZoneRequired, got %v", err)
	}
}

func TestWorkflowValidateValid(t *testing.T) {
	c := dataform.WorkflowConfig{Name: workflowNightly, ReleaseConfigName: releaseProd}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorkflowValidateMissingName(t *testing.T) {
	c := dataform.WorkflowConfig{ReleaseConfigName: releaseProd}
	if err := c.Validate(); !errors.Is(err, dataform.ErrWorkflowNameRequired) {
		t.Errorf("expected ErrWorkflowNameRequired, got %v", err)
	}
}

func TestWorkflowValidateMissingReleaseConfig(t *testing.T) {
	c := dataform.WorkflowConfig{Name: workflowNightly}
	if err := c.Validate(); !errors.Is(err, dataform.ErrWorkflowReleaseConfigRequired) {
		t.Errorf("expected ErrWorkflowReleaseConfigRequired, got %v", err)
	}
}

func TestWorkflowValidateScheduleWithoutTimeZone(t *testing.T) {
	c := dataform.WorkflowConfig{Name: workflowNightly, ReleaseConfigName: releaseProd, CronSchedule: cronDaily}
	if err := c.Validate(); !errors.Is(err, dataform.ErrWorkflowTimeZoneRequired) {
		t.Errorf("expected ErrWorkflowTimeZoneRequired, got %v", err)
	}
}
