package apierr_test

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/platform/apierr"
)

func ExampleIsRetryable() {
	err := fmt.Errorf("GET /employees: %w", apierr.ErrRateLimited)

	if apierr.IsRetryable(err) {
		fmt.Println("back off and retry")
	}
	// Output:
	// back off and retry
}

func ExampleStatusError() {
	// httpclient returns a StatusError.
	err := apierr.NewStatusError(429, "slow down", apierr.ErrRateLimited)

	// Category check with errors.Is.
	if errors.Is(err, apierr.ErrRateLimited) {
		fmt.Println("rate limited")
	}

	// Context extraction with errors.As.
	var se *apierr.StatusError
	if errors.As(err, &se) {
		fmt.Printf("status=%d body=%s\n", se.Status, se.Body)
	}
	// Output:
	// rate limited
	// status=429 body=slow down
}

func ExampleErrUnauthorized() {
	// Vendor wraps apierr sentinels.
	vendorErr := fmt.Errorf("postmark: %w", apierr.ErrUnauthorized)

	if errors.Is(vendorErr, apierr.ErrUnauthorized) {
		fmt.Println("token expired — refresh and retry")
	}
	// Output:
	// token expired — refresh and retry
}
