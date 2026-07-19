// Domain:  Governance
// Concern: Are scope capabilities correctly defined?
package scope_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/scope"
)

func TestOrganizationSupportsClassification(t *testing.T) {
	if !scope.Organization.SupportsClassification() {
		t.Error("Organization scope must support classification")
	}
}

func TestContainerDoesNotSupportClassification(t *testing.T) {
	if scope.Container.SupportsClassification() {
		t.Error("Container scope must not support classification")
	}
}

func TestOrganizationSupportsOrgExport(t *testing.T) {
	if !scope.Organization.SupportsOrgExport() {
		t.Error("Organization scope must support org export")
	}
}

func TestContainerDoesNotSupportOrgExport(t *testing.T) {
	if scope.Container.SupportsOrgExport() {
		t.Error("Container scope must not support org export")
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		scope scope.Scope
		want  string
	}{
		{scope.Organization, "organization"},
		{scope.Container, "container"},
		{scope.Scope(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.scope.String(); got != tt.want {
			t.Errorf("Scope(%d).String() = %q, want %q", tt.scope, got, tt.want)
		}
	}
}
