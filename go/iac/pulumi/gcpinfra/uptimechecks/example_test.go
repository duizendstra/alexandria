package uptimechecks_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/uptimechecks"
)

func ExampleConfig_Validate() {
	// An IAP-protected endpoint: accept the sign-in redirect (3xx) as healthy.
	c := &uptimechecks.Config{
		DisplayName:           "portal",
		Host:                  "portal.example.com",
		AcceptedStatusClasses: []string{uptimechecks.Class2xx, uptimechecks.Class3xx},
	}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}
