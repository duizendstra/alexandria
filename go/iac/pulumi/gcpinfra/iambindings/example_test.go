package iambindings_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/iambindings"
)

func ExampleBinding_Validate() {
	b := iambindings.Binding{Name: "test", ProjectID: "my-project", Role: "roles/viewer", Member: "user:a@b.com"}
	fmt.Println(b.Validate())
	// Output:
	// <nil>
}
