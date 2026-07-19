package classification_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/classification"
)

func ExampleDimension_Validate() {
	d := classification.Dimension{ShortName: "environment", Description: "The environment"}
	fmt.Println(d.Validate())
	// Output:
	// <nil>
}

func ExampleDimension_Validate_error() {
	d := classification.Dimension{Description: "Missing short name"}
	fmt.Println(d.Validate())
	// Output:
	// classification: ShortName is required
}
