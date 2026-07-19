package plan_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/plan"
	"github.com/duizendstra/alexandria/go/governance/scope"
)

const (
	orgParent    = "organizations/123"
	folderParent = "folders/123"
	planRoot     = "example"
	envDev       = "dev"
	envProd      = "prod"
	billingRef   = "BILLING-123"
)

func TestNewStarter(t *testing.T) {
	t.Run("valid folder scope", func(t *testing.T) {
		p, err := plan.NewStarter(scope.Container, folderParent, planRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.Tier != plan.Starter {
			t.Errorf("tier = %v, want Starter", p.Tier)
		}
	})

	t.Run("org scope requires orgID", func(t *testing.T) {
		_, err := plan.NewStarter(scope.Organization, orgParent, planRoot)
		if err == nil {
			t.Fatal("expected error: org scope requires OrgID but Starter cannot provide it")
		}
	})

	t.Run("missing parent", func(t *testing.T) {
		_, err := plan.NewStarter(scope.Container, "", planRoot)
		if err == nil {
			t.Fatal("expected error for missing parent")
		}
	})
}

func TestNewStandard(t *testing.T) {
	t.Run("valid with environments", func(t *testing.T) {
		p, err := plan.NewStandard(scope.Container, folderParent, planRoot, []string{envDev, envProd})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.Tier != plan.Standard {
			t.Errorf("tier = %v, want Standard", p.Tier)
		}

		if len(p.Hierarchy.Children) != 2 {
			t.Errorf("children = %d, want 2", len(p.Hierarchy.Children))
		}
	})

	t.Run("no environments", func(t *testing.T) {
		_, err := plan.NewStandard(scope.Container, folderParent, planRoot, nil)
		if err == nil {
			t.Fatal("expected error for no environments")
		}
	})

	t.Run("duplicate environments", func(t *testing.T) {
		_, err := plan.NewStandard(scope.Container, folderParent, planRoot, []string{envDev, envDev})
		if err == nil {
			t.Fatal("expected error for duplicate environments")
		}
	})
}

func TestNewEnterprise(t *testing.T) {
	dims := []classification.Dimension{
		{ShortName: "environment", Description: "Deployment environment"},
	}

	t.Run("valid", func(t *testing.T) {
		p, err := plan.NewEnterprise(
			orgParent, planRoot, []string{envDev, envProd},
			dims, billingRef, "123",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.Tier != plan.Enterprise {
			t.Errorf("tier = %v, want Enterprise", p.Tier)
		}

		if p.Scope != scope.Organization {
			t.Errorf("scope = %v, want Organization", p.Scope)
		}
	})

	t.Run("missing orgID", func(t *testing.T) {
		_, err := plan.NewEnterprise(
			orgParent, planRoot, []string{envDev},
			dims, billingRef, "",
		)
		if err == nil {
			t.Fatal("expected error for missing orgID")
		}
	})

	t.Run("no environments", func(t *testing.T) {
		_, err := plan.NewEnterprise(
			orgParent, planRoot, nil,
			dims, billingRef, "123",
		)
		if err == nil {
			t.Fatal("expected error for no environments")
		}
	})
}

func TestParseTier(t *testing.T) {
	tests := []struct {
		input string
		want  plan.Tier
		err   bool
	}{
		{"starter", plan.Starter, false},
		{"standard", plan.Standard, false},
		{"enterprise", plan.Enterprise, false},
		{"unknown", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := plan.ParseTier(tt.input)
			if (err != nil) != tt.err {
				t.Fatalf("ParseTier(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			}

			if !tt.err && got != tt.want {
				t.Errorf("ParseTier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTierString(t *testing.T) {
	if plan.Starter.String() != "starter" {
		t.Errorf("Starter.String() = %q", plan.Starter.String())
	}

	if plan.Enterprise.String() != "enterprise" {
		t.Errorf("Enterprise.String() = %q", plan.Enterprise.String())
	}
}
