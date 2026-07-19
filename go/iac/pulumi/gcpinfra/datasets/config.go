package datasets

import "errors"

var (
	// ErrIDRequired means the dataset has no identifier.
	ErrIDRequired = errors.New("datasets: ID is required")
	// ErrFriendlyNameRequired means the dataset has no human-readable name.
	ErrFriendlyNameRequired = errors.New("datasets: FriendlyName is required")
	// ErrLocationRequired means the dataset has no region.
	ErrLocationRequired = errors.New("datasets: Location is required")
)

// Config defines a dataset to be provisioned.
type Config struct {
	// ID is the dataset identifier (e.g. "billing_export").
	ID string `json:"id"`
	// FriendlyName is the human-readable name.
	FriendlyName string `json:"friendlyName"`
	// Description explains the dataset's purpose.
	Description string `json:"description"`
	// Location is the region (e.g. "europe-west4").
	Location string `json:"location"`
	// Labels are key-value pairs for resource organization.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the dataset configuration is complete.
func (c Config) Validate() error {
	if c.ID == "" {
		return ErrIDRequired
	}
	if c.FriendlyName == "" {
		return ErrFriendlyNameRequired
	}
	if c.Location == "" {
		return ErrLocationRequired
	}

	return nil
}
