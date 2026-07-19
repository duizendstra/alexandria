package hierarchy

import (
	"errors"
	"fmt"
)

// Validation errors. Test for them with [errors.Is].
var (
	// ErrParentRequired means the parent container reference is missing.
	ErrParentRequired = errors.New("hierarchy: Parent is required")
	// ErrRootNameRequired means the root container name is missing.
	ErrRootNameRequired = errors.New("hierarchy: RootName is required")
	// ErrEmptyChild means a child container name is empty.
	ErrEmptyChild = errors.New("hierarchy: child name must not be empty")
	// ErrDuplicateChild means two children share a name.
	ErrDuplicateChild = errors.New("hierarchy: duplicate child")
	// ErrNoChildren means at least one child container is required.
	ErrNoChildren = errors.New("hierarchy: at least one child is required")
)

// Config defines a desired organizational container hierarchy.
//
// Parent is an opaque reference to the parent container — its format
// is cloud-specific and validated by the adapter, not here.
// GCP: "organizations/123" or "folders/456"
// AWS: "r-xxxx" (root) or "ou-xxxx-yyyyyyyy".
type Config struct {
	// Parent is the resource path where the root container is created.
	// Format is cloud-specific — validated by the adapter.
	Parent string
	// RootName is the display name of the root container.
	RootName string
	// Children are the child container names under root.
	Children []string
}

// ValidateBase checks that parent and root name are set.
// Called by all tiers.
func (c Config) ValidateBase() error {
	if c.Parent == "" {
		return ErrParentRequired
	}

	if c.RootName == "" {
		return ErrRootNameRequired
	}

	return nil
}

// ValidateChildren checks that children are non-empty and unique.
// Called by Standard and Enterprise tiers.
func (c Config) ValidateChildren() error {
	seen := make(map[string]bool)

	for _, child := range c.Children {
		if child == "" {
			return ErrEmptyChild
		}

		if seen[child] {
			return fmt.Errorf("%w %q", ErrDuplicateChild, child)
		}

		seen[child] = true
	}

	return nil
}

// Validate checks the full hierarchy configuration.
// Equivalent to ValidateBase + requiring at least one child + ValidateChildren.
func (c Config) Validate() error {
	if err := c.ValidateBase(); err != nil {
		return err
	}

	if len(c.Children) == 0 {
		return ErrNoChildren
	}

	return c.ValidateChildren()
}
