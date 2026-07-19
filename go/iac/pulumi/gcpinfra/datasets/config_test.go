package datasets_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
)

func TestValidateValid(t *testing.T) {
	c := datasets.Config{ID: "billing_export", FriendlyName: "Billing Export", Location: "europe-west4"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingID(t *testing.T) {
	c := datasets.Config{FriendlyName: "x", Location: "y"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestValidateMissingFriendlyName(t *testing.T) {
	c := datasets.Config{ID: "x", Location: "y"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing FriendlyName")
	}
}

func TestValidateMissingLocation(t *testing.T) {
	c := datasets.Config{ID: "x", FriendlyName: "y"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing Location")
	}
}
