package gcp_test

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/retry"
	gcp "github.com/duizendstra/alexandria/go/retry/gcp"
	"google.golang.org/api/googleapi"
)

func ExampleWithRetry() {
	ctx := context.Background()

	attempt := 0
	err := gcp.WithRetry(ctx, func() error {
		attempt++
		if attempt < 2 {
			// Simulate a transient Google API rate limit limit error (429)
			return &googleapi.Error{
				Code:    429,
				Message: "Rate limit exceeded",
			}
		}
		fmt.Println("GCP Operation succeeded")
		return nil
	})

	if err != nil {
		fmt.Printf("failed: %v\n", err)
	}

	// Output:
	// GCP Operation succeeded
}

func ExampleClassify() {
	ctx := context.Background()

	// Simulate a permanent 404 error
	err := &googleapi.Error{
		Code:    404,
		Message: "Not found",
	}

	classified := gcp.Classify(ctx, err, 1)

	// Since 404 is a permanent failure, retry.IsPermanent should be true
	if retry.IsPermanent(classified) {
		fmt.Println("Error classified as permanent (fail-fast)")
	} else {
		fmt.Println("Error classified as retryable")
	}

	// Output:
	// Error classified as permanent (fail-fast)
}
