package budgets_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/budgets"
)

func ExampleConfig_Validate() {
	c := &budgets.Config{
		DisplayName:    "Monthly",
		Amount:         100,
		BillingAccount: "X",
		Scope:          "1",
		Thresholds:     []float64{1},
		AlertEmails:    []string{"a@b.com"},
	}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}
