package retry_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/duizendstra/alexandria/go/retry"
)

func ExampleDo() {
	attempt := 0
	err := retry.Do(context.Background(), 3, func() error {
		attempt++
		if attempt < 3 {
			return errors.New("transient issue")
		}
		fmt.Println("Success on attempt 3")
		return nil
	})

	if err != nil {
		fmt.Printf("failed: %v\n", err)
	}

	// Output:
	// Success on attempt 3
}

func ExampleTransport() {
	// Setup a local test server to mimic a transient failure.
	serverAttempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverAttempts++
		if serverAttempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	shouldRetry := func(code int) bool {
		return code == http.StatusServiceUnavailable
	}

	client := &http.Client{
		Transport: retry.Transport(3, shouldRetry, nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		fmt.Printf("failed request creation: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("failed execution: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.StatusCode)

	// Output:
	// Response status: 200
}
