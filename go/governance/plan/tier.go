package plan

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/governance/scope"
)

// Tier represents the governance maturity level.
// Following Google Cloud naming: Starter / Standard / Enterprise.
type Tier int

const (
	// Starter is minimal governance — single folder, no environment
	// separation, no classification, no billing controls.
	Starter Tier = iota

	// Standard provides structured governance with environment separation.
	// Available at both Folder and Organization scope.
	Standard

	// Enterprise provides full governance: classification dimensions,
	// billing controls, and org-level exports. Requires Organization scope.
	Enterprise
)

// Tier names as used in configuration.
const (
	tierNameStarter    = "starter"
	tierNameStandard   = "standard"
	tierNameEnterprise = "enterprise"
)

// String returns the tier name.
func (t Tier) String() string {
	switch t {
	case Starter:
		return tierNameStarter
	case Standard:
		return tierNameStandard
	case Enterprise:
		return tierNameEnterprise
	default:
		return "unknown"
	}
}

// ParseTier converts a string to a Tier.
func ParseTier(s string) (Tier, error) {
	switch s {
	case tierNameStarter:
		return Starter, nil
	case tierNameStandard:
		return Standard, nil
	case tierNameEnterprise:
		return Enterprise, nil
	default:
		return 0, fmt.Errorf("%w %q", ErrUnknownTier, s)
	}
}

// NewStarter builds a minimal governance plan.
// Single root folder, no environment separation.
func NewStarter(s scope.Scope, parent, rootName string) (*Plan, error) {
	p := &Plan{
		Tier:  Starter,
		Scope: s,
		Hierarchy: hierarchy.Config{
			Parent:   parent,
			RootName: rootName,
		},
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return p, nil
}

// NewStandard builds a governance plan with environment separation.
// Requires at least one environment (child folder).
func NewStandard(s scope.Scope, parent, rootName string, environments []string) (*Plan, error) {
	p := &Plan{
		Tier:  Standard,
		Scope: s,
		Hierarchy: hierarchy.Config{
			Parent:   parent,
			RootName: rootName,
			Children: environments,
		},
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return p, nil
}

// NewEnterprise builds a full governance plan with classification,
// billing controls, and org-level exports. Requires Organization scope.
func NewEnterprise(parent, rootName string, environments []string,
	dims []classification.Dimension, billing, orgID string) (*Plan, error) {
	p := &Plan{
		Tier:  Enterprise,
		Scope: scope.Organization,
		Hierarchy: hierarchy.Config{
			Parent:   parent,
			RootName: rootName,
			Children: environments,
		},
		Dimensions:     dims,
		BillingAccount: billing,
		OrgID:          orgID,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return p, nil
}
