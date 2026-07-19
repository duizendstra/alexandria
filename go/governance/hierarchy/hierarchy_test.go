// Domain:  Governance
// Concern: Are all hierarchy invariants enforced?
package hierarchy_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/hierarchy"
)

const (
	orgParent = "organizations/123"
	rootName  = "example"
	childProd = "prod"
)

func TestValidateValid(t *testing.T) {
	cfg := hierarchy.Config{
		Parent:   "organizations/123456789",
		RootName: rootName,
		Children: []string{"shared", "acceptance", "production"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAnyParentFormat(t *testing.T) {
	// Hierarchy accepts any non-empty parent — format is cloud-specific.
	cfg := hierarchy.Config{
		Parent:   "r-a1b2",
		RootName: "acme",
		Children: []string{"dev", childProd},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error for non-GCP parent: %v", err)
	}
}

func TestValidateMissingParent(t *testing.T) {
	cfg := hierarchy.Config{RootName: rootName, Children: []string{childProd}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Parent")
	}
}

func TestValidateMissingRootName(t *testing.T) {
	cfg := hierarchy.Config{Parent: orgParent, Children: []string{childProd}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing RootName")
	}
}

func TestValidateEmptyChildren(t *testing.T) {
	cfg := hierarchy.Config{Parent: orgParent, RootName: rootName}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty children")
	}
}

func TestValidateEmptyChildName(t *testing.T) {
	cfg := hierarchy.Config{Parent: orgParent, RootName: rootName, Children: []string{""}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty child name")
	}
}

func TestValidateDuplicateChild(t *testing.T) {
	cfg := hierarchy.Config{Parent: orgParent, RootName: rootName, Children: []string{childProd, childProd}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for duplicate child")
	}
}
