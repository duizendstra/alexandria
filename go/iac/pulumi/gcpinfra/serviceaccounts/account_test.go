package serviceaccounts_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/serviceaccounts"
)

func TestValidateValid(t *testing.T) {
	a := serviceaccounts.Account{ID: "example-worker-prod", DisplayName: "Example worker prod"}
	if err := a.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingID(t *testing.T) {
	a := serviceaccounts.Account{DisplayName: "test"}
	if err := a.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestValidateMissingDisplayName(t *testing.T) {
	a := serviceaccounts.Account{ID: "test"}
	if err := a.Validate(); err == nil {
		t.Error("expected error for missing DisplayName")
	}
}
