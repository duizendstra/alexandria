package projects_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
)

func ExampleConfig_Validate() {
	cfg := projects.Config{
		Name:           "example-identity",
		FolderID:       "123456789",
		BillingAccount: "012345-6789AB-CDEF01",
		APIs:           []string{"secretmanager.googleapis.com"},
	}
	fmt.Println(cfg.Validate())
	// Output:
	// <nil>
}

func ExampleConfig_Validate_error() {
	cfg := projects.Config{FolderID: "123", BillingAccount: "XXX"}
	fmt.Println(cfg.Validate())
	// Output:
	// projects: Name is required
}
