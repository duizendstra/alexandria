// Domain:  Governance
// Concern: Does the plan enforce all governance policies?
package plan_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/governance/plan"
	"github.com/duizendstra/alexandria/go/governance/scope"
)

func validOrgPlan() *plan.Plan {
	return &plan.Plan{
		Tier:  plan.Enterprise,
		Scope: scope.Organization,
		Hierarchy: hierarchy.Config{
			Parent:   "organizations/123456",
			RootName: planRoot,
			Children: []string{"shared", "production"},
		},
		Dimensions: []classification.Dimension{
			{ShortName: "environment", Description: "The environment"},
		},
		BillingAccount: "AABBCC-112233-DDEEFF",
		OrgID:          "123456",
	}
}

func validContainerPlan() *plan.Plan {
	return &plan.Plan{
		Tier:  plan.Standard,
		Scope: scope.Container,
		Hierarchy: hierarchy.Config{
			Parent:   "folders/987654",
			RootName: planRoot,
			Children: []string{"shared", "production"},
		},
	}
}

func TestValidOrgPlan(t *testing.T) {
	if err := validOrgPlan().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidContainerPlan(t *testing.T) {
	if err := validContainerPlan().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClassificationRequiresEnterprise(t *testing.T) {
	p := validContainerPlan()
	p.Dimensions = []classification.Dimension{
		{ShortName: "env", Description: "env"},
	}
	if err := p.Validate(); err == nil {
		t.Error("expected error: classification at Standard tier")
	}
}

func TestOrgIDRequiredForOrgScope(t *testing.T) {
	p := validOrgPlan()
	p.OrgID = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error: OrgID required at Organization scope")
	}
}

func TestOrgIDForbiddenForContainerScope(t *testing.T) {
	p := validContainerPlan()
	p.OrgID = "should-not-be-here"
	if err := p.Validate(); err == nil {
		t.Error("expected error: OrgID not allowed at Container scope")
	}
}

func TestInvalidHierarchyPropagates(t *testing.T) {
	p := validOrgPlan()
	p.Hierarchy.RootName = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error: invalid hierarchy must propagate")
	}
}

func TestInvalidDimensionPropagates(t *testing.T) {
	p := validOrgPlan()
	p.Dimensions = []classification.Dimension{{ShortName: ""}}
	if err := p.Validate(); err == nil {
		t.Error("expected error: invalid dimension must propagate")
	}
}

func TestEnterprisePlanWithoutDimensions(t *testing.T) {
	p := validOrgPlan()
	p.Dimensions = nil
	if err := p.Validate(); err != nil {
		t.Errorf("enterprise plan without dimensions should be valid: %v", err)
	}
}
