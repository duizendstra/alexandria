package registries_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/registries"
)

func ExampleConfig_Validate() {
	c := registries.Config{ID: "example", Format: "DOCKER", Location: "europe-west4"}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}
