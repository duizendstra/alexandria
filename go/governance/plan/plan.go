package plan

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/governance/scope"
)

// Validation errors. Test for them with [errors.Is].
var (
	// ErrTierViolation means the plan uses a capability its tier does not support.
	ErrTierViolation = errors.New("plan: tier violation")
	// ErrScopeViolation means the plan breaks an organizational scope rule.
	ErrScopeViolation = errors.New("plan: scope violation")
	// ErrUnknownTier means the tier name is not recognized.
	ErrUnknownTier = errors.New("plan: unknown tier")
)

// Plan describes the complete desired governance state.
// Pure data — no provisioning logic. Any tool can consume this.
//
// Use the tier constructors [NewStarter], [NewStandard], or [NewEnterprise]
// to build validated plans with the correct constraints.
type Plan struct {
	// Tier is the governance maturity level.
	Tier Tier

	// Scope determines what capabilities are available.
	Scope scope.Scope

	// Hierarchy defines the organizational container tree.
	Hierarchy hierarchy.Config

	// Dimensions defines classification axes for resources.
	// Empty means skip classification entirely.
	// Only valid for Enterprise tier.
	Dimensions []classification.Dimension

	// BillingAccount is the billing account identifier to export.
	// Empty means not exported to downstream bounded contexts.
	// Only valid for Enterprise tier.
	BillingAccount string

	// OrgID is the organization-level identifier to export.
	// Required when Scope is Organization, empty otherwise.
	OrgID string
}

// Validate checks the plan for structural, tier, and scope consistency.
func (p *Plan) Validate() error {
	// Hierarchy basics: parent and root are always required.
	if err := p.Hierarchy.ValidateBase(); err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	if err := p.validateTier(); err != nil {
		return err
	}

	return p.validateScope()
}

// validateTier dispatches to the tier-specific rule set.
func (p *Plan) validateTier() error {
	switch p.Tier {
	case Starter:
		return p.validateStarter()
	case Standard:
		return p.validateStandard()
	case Enterprise:
		return p.validateEnterprise()
	default:
		return nil
	}
}

func (p *Plan) validateStarter() error {
	if len(p.Hierarchy.Children) > 0 {
		return fmt.Errorf("%w: Starter tier does not support environment folders", ErrTierViolation)
	}

	if len(p.Dimensions) > 0 {
		return fmt.Errorf("%w: Starter tier does not support classification", ErrTierViolation)
	}

	if p.BillingAccount != "" {
		return fmt.Errorf("%w: Starter tier does not support billing export", ErrTierViolation)
	}

	return nil
}

func (p *Plan) validateStandard() error {
	if len(p.Hierarchy.Children) == 0 {
		return fmt.Errorf("%w: Standard tier requires at least one environment", ErrTierViolation)
	}

	if err := p.Hierarchy.ValidateChildren(); err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	if len(p.Dimensions) > 0 {
		return fmt.Errorf("%w: Standard tier does not support classification", ErrTierViolation)
	}

	if p.BillingAccount != "" {
		return fmt.Errorf("%w: Standard tier does not support billing export", ErrTierViolation)
	}

	return nil
}

func (p *Plan) validateEnterprise() error {
	if p.Scope != scope.Organization {
		return fmt.Errorf("%w: Enterprise tier requires Organization scope", ErrTierViolation)
	}

	if len(p.Hierarchy.Children) == 0 {
		return fmt.Errorf("%w: Enterprise tier requires at least one environment", ErrTierViolation)
	}

	if err := p.Hierarchy.ValidateChildren(); err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	if len(p.Dimensions) > 0 {
		if err := classification.ValidateAll(p.Dimensions); err != nil {
			return fmt.Errorf("plan: %w", err)
		}
	}

	return nil
}

// validateScope enforces org-level export rules.
func (p *Plan) validateScope() error {
	if p.Scope.SupportsOrgExport() && p.OrgID == "" {
		return fmt.Errorf("%w: OrgID is required for Organization scope", ErrScopeViolation)
	}

	if !p.Scope.SupportsOrgExport() && p.OrgID != "" {
		return fmt.Errorf("%w: OrgID must be empty for Container scope", ErrScopeViolation)
	}

	return nil
}
