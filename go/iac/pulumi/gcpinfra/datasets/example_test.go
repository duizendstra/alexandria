package datasets_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
)

func ExampleConfig_Validate() {
	c := datasets.Config{ID: "billing_export", FriendlyName: "Billing Export", Location: "europe-west4"}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}
