package dataform

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// ErrNameRequired means the repository has no identifier.
	ErrNameRequired = errors.New("dataform: Name is required")
	// ErrRegionRequired means the repository has no region.
	ErrRegionRequired = errors.New("dataform: Region is required")
	// ErrGitURLRequired means the repository has no Git remote URL.
	ErrGitURLRequired = errors.New("dataform: GitURL is required")
	// ErrDefaultBranchRequired means the repository has no sync branch.
	ErrDefaultBranchRequired = errors.New("dataform: DefaultBranch is required")
	// ErrTokenSecretVersionRequired means the repository has no Git auth secret.
	ErrTokenSecretVersionRequired = errors.New("dataform: TokenSecretVersion is required")
	// ErrReleaseNameRequired means the release config has no identifier.
	ErrReleaseNameRequired = errors.New("dataform: release Name is required")
	// ErrReleaseGitCommitishRequired means the release config has no Git ref.
	ErrReleaseGitCommitishRequired = errors.New("dataform: release GitCommitish is required")
	// ErrReleaseTimeZoneRequired means the release config schedules without a timezone.
	ErrReleaseTimeZoneRequired = errors.New("dataform: release has CronSchedule but no TimeZone")
	// ErrWorkflowNameRequired means the workflow config has no identifier.
	ErrWorkflowNameRequired = errors.New("dataform: workflow Name is required")
	// ErrWorkflowReleaseConfigRequired means the workflow config has no release config.
	ErrWorkflowReleaseConfigRequired = errors.New("dataform: workflow ReleaseConfigName is required")
	// ErrWorkflowTimeZoneRequired means the workflow config schedules without a timezone.
	ErrWorkflowTimeZoneRequired = errors.New("dataform: workflow has CronSchedule but no TimeZone")
)

// RepositoryConfig defines a Dataform repository to be provisioned.
type RepositoryConfig struct {
	// Name is the Pulumi resource name and Dataform repo name.
	Name string
	// DisplayName is the human-readable display name.
	DisplayName string
	// Region is the GCP region (e.g. "europe-west4").
	Region string
	// GitURL is the remote Git repository URL.
	GitURL string
	// DefaultBranch is the branch Dataform syncs from (e.g. "main").
	DefaultBranch string
	// TokenSecretVersion is the full secret version path for Git auth.
	TokenSecretVersion string
	// ForceDelete allows deleting the repository even if it contains nested resources.
	ForceDelete bool
	// Labels are resource labels applied to the repository.
	Labels map[string]string
}

// Validate checks that the configuration is complete.
func (c *RepositoryConfig) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Region == "" {
		return ErrRegionRequired
	}
	if c.GitURL == "" {
		return ErrGitURLRequired
	}
	if c.DefaultBranch == "" {
		return ErrDefaultBranchRequired
	}
	if c.TokenSecretVersion == "" {
		return ErrTokenSecretVersionRequired
	}

	return nil
}

// ReleaseConfig defines a Dataform release configuration.
// DefaultDatabase and Vars accept Pulumi inputs so they can be wired
// from dynamic stack reference outputs (e.g. project IDs per environment).
type ReleaseConfig struct {
	// Name is the release config ID.
	Name string
	// GitCommitish is the branch, tag, or SHA to compile from.
	GitCommitish string
	// DefaultDatabase overrides the default project for compilation.
	DefaultDatabase pulumi.StringInput
	// Vars are compilation variable overrides.
	Vars pulumi.StringMap
	// CronSchedule is an optional cron expression for periodic compilation.
	CronSchedule string
	// TimeZone for the cron schedule (e.g. "Europe/Amsterdam").
	TimeZone string
}

// Validate checks that the release configuration is complete.
func (c *ReleaseConfig) Validate() error {
	if c.Name == "" {
		return ErrReleaseNameRequired
	}
	if c.GitCommitish == "" {
		return ErrReleaseGitCommitishRequired
	}
	if c.CronSchedule != "" && c.TimeZone == "" {
		return fmt.Errorf("%w: %s", ErrReleaseTimeZoneRequired, c.Name)
	}

	return nil
}

// WorkflowConfig defines a Dataform workflow configuration.
type WorkflowConfig struct {
	// Name is the workflow config ID.
	Name string
	// ReleaseConfigName is the release config this workflow executes.
	// Apply constructs the full resource path automatically.
	ReleaseConfigName string
	// CronSchedule is an optional cron expression (empty = manual only).
	CronSchedule string
	// TimeZone for the cron schedule (e.g. "Europe/Amsterdam").
	TimeZone string
	// IncludedTags filters which actions to run (empty = all).
	IncludedTags []string
	// IncludeTransitiveDeps includes all transitive dependencies of tagged actions.
	IncludeTransitiveDeps bool
}

// Validate checks that the workflow configuration is complete.
func (c *WorkflowConfig) Validate() error {
	if c.Name == "" {
		return ErrWorkflowNameRequired
	}
	if c.ReleaseConfigName == "" {
		return ErrWorkflowReleaseConfigRequired
	}
	if c.CronSchedule != "" && c.TimeZone == "" {
		return fmt.Errorf("%w: %s", ErrWorkflowTimeZoneRequired, c.Name)
	}

	return nil
}
