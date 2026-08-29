package gcp_test

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/platform/retry"
	gcp "github.com/duizendstra/alexandria/go/platform/retry/gcp"
	"google.golang.org/api/googleapi"
)

func ExampleWithRetry() {
	ctx := context.Background()

	attempt := 0
	err := gcp.WithRetry(ctx, func() error {
		attempt++
		if attempt < 2 {
			// Simulate a transient Google API rate limit error (429).
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

	// Simulate a permanent 404 error.
	err := &googleapi.Error{
		Code:    404,
		Message: "Not found",
	}

	classified := gcp.Classify(ctx, err, 1)

	// Since 404 is a permanent failure, retry.IsPermanent should be true.
	if retry.IsPermanent(classified) {
		fmt.Println("Error classified as permanent (fail-fast)")
	} else {
		fmt.Println("Error classified as retryable")
	}

	// Output:
	// Error classified as permanent (fail-fast)
}

func ExampleWithRetryVal() {
	ctx := context.Background()

	type DriveFile struct {
		ID   string
		Name string
	}

	file, err := gcp.WithRetryVal(ctx, func() (*DriveFile, error) {
		return &DriveFile{ID: "file-999", Name: "Quarterly Report.pdf"}, nil
	})
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		return
	}

	fmt.Printf("Fetched file: %s (%s)\n", file.Name, file.ID)

	// Output:
	// Fetched file: Quarterly Report.pdf (file-999)
}
