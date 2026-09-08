package secrets_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
)

func ExampleSecret_Validate() {
	s := secrets.Secret{Name: secretName, Value: "secret"}
	fmt.Println(s.Validate())
	// Output:
	// <nil>
}
