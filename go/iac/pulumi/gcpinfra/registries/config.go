package registries

import "errors"

var (
	// ErrIDRequired means the registry has no repository identifier.
	ErrIDRequired = errors.New("registries: ID is required")
	// ErrFormatRequired means the registry has no repository format.
	ErrFormatRequired = errors.New("registries: Format is required")
	// ErrLocationRequired means the registry has no region.
	ErrLocationRequired = errors.New("registries: Location is required")
)

// Config defines a container registry to be provisioned.
type Config struct {
	// ID is the repository identifier (e.g. "example").
	ID string `json:"id"`
	// Description is the human-readable description.
	Description string `json:"description"`
	// Format is the repository format (e.g. "DOCKER").
	Format string `json:"format"`
	// Location is the region (e.g. "europe-west4").
	Location string `json:"location"`
}

// Validate checks that the registry configuration is complete.
func (c Config) Validate() error {
	if c.ID == "" {
		return ErrIDRequired
	}
	if c.Format == "" {
		return ErrFormatRequired
	}
	if c.Location == "" {
		return ErrLocationRequired
	}

	return nil
}
