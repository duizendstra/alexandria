package projects

import (
	"errors"
	"fmt"
)

var (
	// ErrNameRequired means the project has no name.
	ErrNameRequired = errors.New("projects: Name is required")
	// ErrFolderIDRequired means the project has no parent folder.
	ErrFolderIDRequired = errors.New("projects: FolderID is required")
	// ErrBillingAccountRequired means the project has no billing account.
	ErrBillingAccountRequired = errors.New("projects: BillingAccount is required")
	// ErrEmptyAPI means an API name in the enablement list is empty.
	ErrEmptyAPI = errors.New("projects: API name must not be empty")
	// ErrDuplicateAPI means an API appears twice in the enablement list.
	ErrDuplicateAPI = errors.New("projects: duplicate API")
)

// Config defines a project to be provisioned.
type Config struct {
	// Name is the project display name and ID.
	Name string
	// FolderID is the numeric ID of the parent folder.
	FolderID string
	// BillingAccount is the billing account ID (e.g. "012345-6789AB-CDEF01").
	BillingAccount string
	// APIs is the list of APIs to enable on the project.
	APIs []string
}

// Validate checks that the configuration is complete and consistent.
func (c Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.FolderID == "" {
		return ErrFolderIDRequired
	}
	if c.BillingAccount == "" {
		return ErrBillingAccountRequired
	}
	seen := make(map[string]bool)
	for _, api := range c.APIs {
		if api == "" {
			return ErrEmptyAPI
		}
		if seen[api] {
			return fmt.Errorf("%w %q", ErrDuplicateAPI, api)
		}
		seen[api] = true
	}

	return nil
}
