package search_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

func ExampleKind() {
	kinds := []search.Kind{
		search.KindCode,
		search.KindDoc,
		search.KindMemory,
	}
	for _, k := range kinds {
		fmt.Println(k)
	}
	// Output:
	// code
	// doc
	// memory
}
