package plan_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/plan"
	"github.com/duizendstra/alexandria/go/governance/scope"
)

func ExampleNewStandard() {
	p, err := plan.NewStandard(scope.Container, "folders/987", "example", []string{"dev", "prod"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("tier=%s scope=%s envs=%d\n", p.Tier, p.Scope, len(p.Hierarchy.Children))
	// Output:
	// tier=standard scope=container envs=2
}
