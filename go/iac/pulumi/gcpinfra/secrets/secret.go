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
