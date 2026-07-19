package classification

import (
	"errors"
	"fmt"
)

// Validation errors. Test for them with [errors.Is].
var (
	// ErrShortNameRequired means a dimension has no ShortName.
	ErrShortNameRequired = errors.New("classification: ShortName is required")
	// ErrDescriptionRequired means a dimension has no Description.
	ErrDescriptionRequired = errors.New("classification: Description is required")
	// ErrNoDimensions means an empty dimension set was validated.
	ErrNoDimensions = errors.New("classification: at least one dimension is required")
	// ErrDuplicateShortName means two dimensions share a ShortName.
	ErrDuplicateShortName = errors.New("classification: duplicate ShortName")
)

// Dimension defines a classification axis for resources.
// GCP calls these "tag keys", AWS and Azure call them "tags".
// The concept is universal: a named dimension with a description.
type Dimension struct {
	// ShortName is the dimension identifier (e.g. "environment").
	ShortName string `json:"shortName"`
	// Description explains what this dimension classifies.
	Description string `json:"description"`
}

// Validate checks that the dimension definition is complete.
func (d Dimension) Validate() error {
	if d.ShortName == "" {
		return ErrShortNameRequired
	}

	if d.Description == "" {
		return ErrDescriptionRequired
	}

	return nil
}

// ValidateAll checks a slice of dimensions for completeness and uniqueness.
func ValidateAll(dims []Dimension) error {
	if len(dims) == 0 {
		return ErrNoDimensions
	}

	seen := make(map[string]bool)

	for _, d := range dims {
		if err := d.Validate(); err != nil {
			return err
		}

		if seen[d.ShortName] {
			return fmt.Errorf("%w %q", ErrDuplicateShortName, d.ShortName)
		}

		seen[d.ShortName] = true
	}

	return nil
}
