package secrets_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
)

func TestValidateValid(t *testing.T) {
	s := secrets.Secret{Name: "api-key", Value: "secret123"}
	if err := s.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	s := secrets.Secret{Value: "val"}
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing Name")
	}
}

func TestValidateMissingValue(t *testing.T) {
	s := secrets.Secret{Name: "key"}
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing Value")
	}
}

func TestValidateAllValid(t *testing.T) {
	ss := []secrets.Secret{{Name: "a", Value: "1"}, {Name: "b", Value: "2"}}
	if err := secrets.ValidateAll(ss); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAllEmpty(t *testing.T) {
	if err := secrets.ValidateAll(nil); err == nil {
		t.Error("expected error for empty secrets")
	}
}

func TestValidateAllDuplicate(t *testing.T) {
	ss := []secrets.Secret{{Name: "a", Value: "1"}, {Name: "a", Value: "2"}}
	if err := secrets.ValidateAll(ss); err == nil {
		t.Error("expected error for duplicate Name")
	}
}
