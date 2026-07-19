package hierarchy_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/hierarchy"
)

func ExampleConfig_Validate() {
	cfg := hierarchy.Config{
		Parent:   "organizations/123456789",
		RootName: "example",
		Children: []string{"shared", "production"},
	}
	fmt.Println(cfg.Validate())
	// Output:
	// <nil>
}
