package iambindings_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/iambindings"
)

func TestValidateValid(t *testing.T) {
	b := iambindings.Binding{
		Name:      "test",
		ProjectID: "my-project",
		Role:      "roles/viewer",
		Member:    "serviceAccount:sa@project.iam.gserviceaccount.com",
	}
	if err := b.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	b := iambindings.Binding{ProjectID: "p", Role: "r", Member: "m"}
	if err := b.Validate(); err == nil {
		t.Error("expected error for missing Name")
	}
}

func TestValidateMissingProjectID(t *testing.T) {
	b := iambindings.Binding{Name: "n", Role: "r", Member: "m"}
	if err := b.Validate(); err == nil {
		t.Error("expected error for missing ProjectID")
	}
}

func TestValidateMissingRole(t *testing.T) {
	b := iambindings.Binding{Name: "n", ProjectID: "p", Member: "m"}
	if err := b.Validate(); err == nil {
		t.Error("expected error for missing Role")
	}
}

func TestValidateMissingMember(t *testing.T) {
	b := iambindings.Binding{Name: "n", ProjectID: "p", Role: "r"}
	if err := b.Validate(); err == nil {
		t.Error("expected error for missing Member")
	}
}
