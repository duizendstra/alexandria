package privacyfilter_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/discovery/privacyfilter"
	"github.com/duizendstra/alexandria/go/discovery/search"
)

func ExampleFilter_Apply() {
	f := privacyfilter.New()

	docs := []search.Document{
		{ID: "1", Path: "main.go", Content: "package main"},
		{ID: "2", Path: ".env", Content: "SECRET=abc"},
		{ID: "3", Path: "config/credentials.json", Content: "{}"},
	}

	clean, skipped := f.Apply(docs)
	fmt.Printf("Clean: %d, Skipped: %d\n", len(clean), skipped)
	// Output: Clean: 1, Skipped: 2
}
