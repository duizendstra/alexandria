// Domain:  Governance
// Concern: Are all dimension invariants enforced?
package classification_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/classification"
)

const (
	shortEnv = "environment"
	descEnv  = "The environment"
)

func TestValidateValid(t *testing.T) {
	d := classification.Dimension{ShortName: shortEnv, Description: descEnv}
	if err := d.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingShortName(t *testing.T) {
	d := classification.Dimension{Description: descEnv}
	if err := d.Validate(); err == nil {
		t.Error("expected error for missing ShortName")
	}
}

func TestValidateMissingDescription(t *testing.T) {
	d := classification.Dimension{ShortName: shortEnv}
	if err := d.Validate(); err == nil {
		t.Error("expected error for missing Description")
	}
}

func TestValidateAllValid(t *testing.T) {
	dims := []classification.Dimension{
		{ShortName: shortEnv, Description: "env"},
		{ShortName: "managed-by", Description: "tool"},
	}
	if err := classification.ValidateAll(dims); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAllEmpty(t *testing.T) {
	if err := classification.ValidateAll(nil); err == nil {
		t.Error("expected error for empty dimensions")
	}
}

func TestValidateAllDuplicate(t *testing.T) {
	dims := []classification.Dimension{
		{ShortName: "dup", Description: "a"},
		{ShortName: "dup", Description: "b"},
	}
	if err := classification.ValidateAll(dims); err == nil {
		t.Error("expected error for duplicate ShortName")
	}
}
