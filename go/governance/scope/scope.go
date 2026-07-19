package scope

// Scope represents the organizational level at which governance operates.
type Scope int

const (
	// Organization means full org-level governance — hierarchy,
	// classification, and org-level exports are all available.
	Organization Scope = iota

	// Container means sub-org governance — only hierarchy management
	// is available. Classification and org-level exports are not supported.
	Container
)

// SupportsClassification reports whether this scope can manage
// resource classification dimensions (tag keys, labels, etc.).
func (s Scope) SupportsClassification() bool {
	return s == Organization
}

// SupportsOrgExport reports whether this scope publishes
// an organization-level identifier to downstream bounded contexts.
func (s Scope) SupportsOrgExport() bool {
	return s == Organization
}

// String returns a human-readable representation.
func (s Scope) String() string {
	switch s {
	case Organization:
		return "organization"
	case Container:
		return "container"
	default:
		return "unknown"
	}
}
