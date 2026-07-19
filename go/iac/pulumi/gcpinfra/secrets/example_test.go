package secrets_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
)

func ExampleSecret_Validate() {
	s := secrets.Secret{Name: "api-key", Value: "secret"}
	fmt.Println(s.Validate())
	// Output:
	// <nil>
}
