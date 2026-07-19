package scope_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/scope"
)

func ExampleScope_SupportsClassification() {
	fmt.Println(scope.Organization.SupportsClassification())
	fmt.Println(scope.Container.SupportsClassification())
	// Output:
	// true
	// false
}
