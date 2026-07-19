package logsinks_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/logsinks"
)

func ExampleConfig_Validate() {
	c := logsinks.Config{Name: "org-audit", OrgID: "123"}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}
