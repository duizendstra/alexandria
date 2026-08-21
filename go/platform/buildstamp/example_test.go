package buildstamp_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/platform/buildstamp"
)

const releaseSHA = "1111111111111111111111111111111111111111"

// A supervising script reads the binary's own version line and decides whether
// it may run.
func ExampleParseStamp() {
	line := "tool 1.4.0 commit=" + releaseSHA + " dirty=false built=2026-01-02T15:04:05Z lib=abc1234"

	stamp, err := buildstamp.ParseStamp(line)
	if err != nil {
		fmt.Println("unreadable stamp:", err)
		return
	}
	fmt.Println(stamp.Name, stamp.Short())

	if err := stamp.Matches(releaseSHA); err != nil {
		fmt.Println("refused:", err)
		return
	}
	fmt.Println("accepted")
	// Output:
	// tool 1.4.0 (1111111)
	// accepted
}

// A dirty build is refused with a reason the operator can act on.
func ExampleStamp_Matches() {
	stamp := buildstamp.Stamp{Commit: releaseSHA, Dirty: true}
	fmt.Println(stamp.Matches(releaseSHA))
	// Output: binary was built from a dirty working tree — rebuild from a clean checkout
}
