package serviceaccounts_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/serviceaccounts"
)

func ExampleAccount_Validate() {
	a := serviceaccounts.Account{ID: "example-worker-prod", DisplayName: "Example worker prod"}
	fmt.Println(a.Validate())
	// Output:
	// <nil>
}
