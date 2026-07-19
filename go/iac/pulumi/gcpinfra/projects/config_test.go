package projects_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
)

const (
	testProjectName = "example-identity"
	testShortName   = "test"
	testFolderID    = "123"
	testBilling     = "XXX"
)

func TestValidateValid(t *testing.T) {
	cfg := projects.Config{
		Name:           testProjectName,
		FolderID:       "123456789",
		BillingAccount: "012345-6789AB-CDEF01",
		APIs:           []string{"secretmanager.googleapis.com"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateNoAPIs(t *testing.T) {
	cfg := projects.Config{
		Name:           testProjectName,
		FolderID:       testFolderID,
		BillingAccount: testBilling,
	}
	if err := cfg.Validate(); err != nil {
		t.Error("expected no error for empty APIs — projects can have zero APIs")
	}
}

func TestValidateMissingName(t *testing.T) {
	cfg := projects.Config{FolderID: testFolderID, BillingAccount: testBilling}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Name")
	}
}

func TestValidateMissingFolderID(t *testing.T) {
	cfg := projects.Config{Name: testShortName, BillingAccount: testBilling}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing FolderID")
	}
}

func TestValidateMissingBilling(t *testing.T) {
	cfg := projects.Config{Name: testShortName, FolderID: testFolderID}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing BillingAccount")
	}
}

func TestValidateDuplicateAPI(t *testing.T) {
	cfg := projects.Config{
		Name:           testShortName,
		FolderID:       testFolderID,
		BillingAccount: testBilling,
		APIs:           []string{"foo.googleapis.com", "foo.googleapis.com"},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for duplicate API")
	}
}

func TestValidateEmptyAPI(t *testing.T) {
	cfg := projects.Config{
		Name:           testShortName,
		FolderID:       testFolderID,
		BillingAccount: testBilling,
		APIs:           []string{""},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty API name")
	}
}
