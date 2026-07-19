package logsinks

import "errors"

var (
	// ErrNameRequired means the log sink has no identifier.
	ErrNameRequired = errors.New("logsinks: Name is required")
	// ErrOrgIDRequired means the log sink has no source organization.
	ErrOrgIDRequired = errors.New("logsinks: OrgID is required")
)

// Config defines an org-level log sink.
type Config struct {
	// Name is the sink identifier.
	Name string `json:"name"`
	// OrgID is the organization to collect logs from.
	OrgID string `json:"orgId"`
	// Filter is the Cloud Logging filter expression.
	Filter string `json:"filter"`
	// IncludeChildren includes logs from all sub-resources.
	IncludeChildren bool `json:"includeChildren"`
}

// Validate checks that the log sink configuration is complete.
func (c Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.OrgID == "" {
		return ErrOrgIDRequired
	}

	return nil
}
