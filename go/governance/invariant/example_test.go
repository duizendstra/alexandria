package invariant_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/invariant"
)

// A rule reports evidence whether it passes or fails, so the result is
// readable without rerunning it.
func ExampleBuilder() {
	b := invariant.New("row-count-matches", 1)
	b.Notef("source=1200 target=1200")
	check := b.Done()

	fmt.Println(check.Status, check.Evidence[0])
	// Output: PASS source=1200 target=1200
}

// An anomaly means "a human should look", not "this is broken" — and a real
// failure always outranks it.
func ExampleBuilder_Anomalyf() {
	b := invariant.New("permissions-readable", 2)
	b.Anomalyf("could not read ACL for 3 items")
	fmt.Println(b.Status())

	b.Failf("2 items still owned by the source")
	fmt.Println(b.Status())
	// Output:
	// ANOMALY
	// FAIL
}

// Rules collected in a slice make the suite enumerable: countable, listable,
// and checkable for whether each one has its own test.
func ExampleEvaluate() {
	type report struct{ moved, expected int }

	rules := []invariant.Rule[report]{
		{Name: "everything-moved", Invariant: 1, Eval: func(r report) invariant.Check {
			b := invariant.New("everything-moved", 1)
			if r.moved != r.expected {
				b.Failf("moved %d of %d", r.moved, r.expected)
			} else {
				b.Notef("moved all %d", r.expected)
			}
			return b.Done()
		}},
	}

	checks := invariant.Evaluate(report{moved: 8, expected: 10}, rules)
	fmt.Println(invariant.Verdict(checks), invariant.Failed(checks))
	// Output: FAIL [everything-moved]
}
