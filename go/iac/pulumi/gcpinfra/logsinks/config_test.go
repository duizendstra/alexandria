package logsinks_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/logsinks"
)

func TestValidateValid(t *testing.T) {
	c := logsinks.Config{Name: "org-audit", OrgID: "123"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	c := logsinks.Config{OrgID: "123"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing Name")
	}
}

func TestValidateMissingOrgID(t *testing.T) {
	c := logsinks.Config{Name: "sink"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing OrgID")
	}
}
