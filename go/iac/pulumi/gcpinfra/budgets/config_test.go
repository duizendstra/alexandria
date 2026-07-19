package budgets_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/budgets"
)

func TestValidateValid(t *testing.T) {
	c := budgets.Config{
		DisplayName:    "Org Monthly",
		Amount:         100,
		Currency:       "USD",
		BillingAccount: "XXX",
		Scope:          "123",
		Thresholds:     []float64{0.50, 0.75, 1.00},
		AlertEmails:    []string{"admin@example.com"},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateZeroAmount(t *testing.T) {
	c := budgets.Config{
		DisplayName: "x", BillingAccount: "x", Scope: "x",
		Thresholds: []float64{1}, AlertEmails: []string{"a"},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestValidateMissingThresholds(t *testing.T) {
	c := budgets.Config{
		DisplayName: "x", Amount: 1, BillingAccount: "x",
		Scope: "x", AlertEmails: []string{"a"},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing thresholds")
	}
}

func TestValidateMissingAlertEmails(t *testing.T) {
	c := budgets.Config{
		DisplayName: "x", Amount: 1, BillingAccount: "x",
		Scope: "x", Thresholds: []float64{1},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing alert emails")
	}
}
