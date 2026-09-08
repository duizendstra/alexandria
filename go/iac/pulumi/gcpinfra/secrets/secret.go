package secrets

import (
	"errors"
	"fmt"
)

var (
	// ErrNameRequired means the secret has no identifier.
	ErrNameRequired = errors.New("secrets: Name is required")
	// ErrValueRequired means the secret has no data.
	ErrValueRequired = errors.New("secrets: Value is required")
	// ErrNoSecrets means the secret list is empty.
	ErrNoSecrets = errors.New("secrets: at least one secret is required")
	// ErrDuplicateName means two secrets share a name.
	ErrDuplicateName = errors.New("secrets: duplicate Name")
	// ErrSecretDataType means the secret value could not be marked as a
	// Pulumi secret output.
	ErrSecretDataType = errors.New("secrets: secret data is not a string output")
	// ErrNoContainers means the container list is empty.
	ErrNoContainers = errors.New("secrets: at least one container is required")
)

// Secret defines a managed secret with its initial value source.
type Secret struct {
	// Name is the secret identifier.
	Name string `json:"name"`
	// Value is the secret data. Sourced at deploy time by the caller.
	Value string `json:"-"`
}

// Validate checks that the secret definition is complete.
func (s Secret) Validate() error {
	if s.Name == "" {
		return ErrNameRequired
	}
	if s.Value == "" {
		return ErrValueRequired
	}

	return nil
}

// ValidateAll checks a slice of secrets for completeness and uniqueness.
func ValidateAll(secrets []Secret) error {
	if len(secrets) == 0 {
		return ErrNoSecrets
	}
	seen := make(map[string]bool)
	for _, s := range secrets {
		if err := s.Validate(); err != nil {
			return err
		}
		if seen[s.Name] {
			return fmt.Errorf("%w %q", ErrDuplicateName, s.Name)
		}
		seen[s.Name] = true
	}

	return nil
}

// Container defines a secret whose value this stack must never hold.
//
// Secret is the wrong shape whenever the value belongs to somebody the stack
// cannot ask: a key issued by hand, a credential a third party rotates on its
// own schedule, a password only a key holder is allowed to see. Passing such a
// value to Secret would write it into the state file, and reading it back out
// of the state file is exactly the exposure the arrangement exists to avoid.
//
// A container therefore has no Value: ApplyContainers creates the secret and
// stops. Until somebody adds a version out of band the secret has none, and
// every read of it fails — which is the honest failure, and a far better one
// than a stack that quietly holds the value it promised not to.
type Container struct {
	// Name is the secret identifier.
	Name string `json:"name"`
	// Labels are applied to the secret. Optional.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the container definition is complete.
func (c Container) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}

	return nil
}

// ValidateContainers checks a slice of containers for completeness and
// uniqueness.
func ValidateContainers(containers []Container) error {
	if len(containers) == 0 {
		return ErrNoContainers
	}

	seen := make(map[string]bool, len(containers))

	for _, c := range containers {
		if err := c.Validate(); err != nil {
			return err
		}

		if seen[c.Name] {
			return fmt.Errorf("%w %q", ErrDuplicateName, c.Name)
		}

		seen[c.Name] = true
	}

	return nil
}
