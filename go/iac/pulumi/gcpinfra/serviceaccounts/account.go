package serviceaccounts

import "errors"

var (
	// ErrIDRequired means the account has no identifier.
	ErrIDRequired = errors.New("serviceaccounts: ID is required")
	// ErrDisplayNameRequired means the account has no display name.
	ErrDisplayNameRequired = errors.New("serviceaccounts: DisplayName is required")
)

// Account defines a service account to be provisioned.
type Account struct {
	// ID is the unique account identifier (e.g. "example-worker-prod").
	ID string `json:"id"`
	// DisplayName is the human-readable description.
	DisplayName string `json:"displayName"`
}

// Validate checks that the account definition is complete.
func (a Account) Validate() error {
	if a.ID == "" {
		return ErrIDRequired
	}
	if a.DisplayName == "" {
		return ErrDisplayNameRequired
	}

	return nil
}
