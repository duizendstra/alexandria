package apierr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/apierr"
)

func TestSentinels_AreDistinct(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		apierr.ErrInvalidInput,
		apierr.ErrUnauthorized,
		apierr.ErrForbidden,
		apierr.ErrNotFound,
		apierr.ErrTimeout,
		apierr.ErrConflict,
		apierr.ErrRateLimited,
		apierr.ErrServerError,
		apierr.ErrUnexpectedStatus,
		apierr.ErrRetriesExceeded,
		apierr.ErrAuthFailed,
		apierr.ErrAPIError,
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %d (%v) matches sentinel %d (%v)", i, a, j, b)
			}
		}
	}
}

func TestSentinels_WrappedIsMatch(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("postmark: %w", apierr.ErrRateLimited)

	if !errors.Is(wrapped, apierr.ErrRateLimited) {
		t.Error("wrapped error should match ErrRateLimited")
	}

	if errors.Is(wrapped, apierr.ErrNotFound) {
		t.Error("wrapped error should not match ErrNotFound")
	}
}

func TestSentinels_VendorWrapping(t *testing.T) {
	t.Parallel()

	// Vendor defines its own sentinel by wrapping apierr.
	vendorErr := fmt.Errorf("postmark: %w", apierr.ErrRateLimited)

	// Wrap it again in context.
	callErr := fmt.Errorf("SendEmail: %w", vendorErr)

	// Both levels should match.
	if !errors.Is(callErr, apierr.ErrRateLimited) {
		t.Error("double-wrapped error should match apierr.ErrRateLimited")
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	retryable := []error{
		apierr.ErrRateLimited,
		apierr.ErrServerError,
		apierr.ErrTimeout,
		fmt.Errorf("vendor: %w", apierr.ErrRateLimited),
	}

	for _, err := range retryable {
		if !apierr.IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = false, want true", err)
		}
	}

	notRetryable := []error{
		apierr.ErrInvalidInput,
		apierr.ErrUnauthorized,
		apierr.ErrForbidden,
		apierr.ErrNotFound,
		apierr.ErrConflict,
		apierr.ErrUnexpectedStatus,
		apierr.ErrRetriesExceeded,
		apierr.ErrAuthFailed,
		apierr.ErrAPIError,
	}

	for _, err := range notRetryable {
		if apierr.IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = true, want false", err)
		}
	}
}

func TestStatusError_ErrorMessage(t *testing.T) {
	t.Parallel()

	se := apierr.NewStatusError(429, "slow down", apierr.ErrRateLimited)

	want := "rate limited: 429 slow down"
	if se.Error() != want {
		t.Errorf("Error() = %q, want %q", se.Error(), want)
	}
}

func TestStatusError_ErrorMessage_NoBody(t *testing.T) {
	t.Parallel()

	se := apierr.NewStatusError(403, "", apierr.ErrForbidden)

	want := "forbidden: 403"
	if se.Error() != want {
		t.Errorf("Error() = %q, want %q", se.Error(), want)
	}
}

func TestStatusError_Unwrap(t *testing.T) {
	t.Parallel()

	se := apierr.NewStatusError(404, "", apierr.ErrNotFound)

	if !errors.Is(se, apierr.ErrNotFound) {
		t.Error("StatusError should unwrap to ErrNotFound")
	}

	if errors.Is(se, apierr.ErrForbidden) {
		t.Error("StatusError should not match ErrForbidden")
	}
}

func TestStatusError_ErrorsAs(t *testing.T) {
	t.Parallel()

	original := apierr.NewStatusError(429, "too fast", apierr.ErrRateLimited)
	wrapped := fmt.Errorf("GET /employees: %w", original)

	var se *apierr.StatusError
	if !errors.As(wrapped, &se) {
		t.Fatal("errors.As should extract StatusError")
	}

	if se.Status != 429 {
		t.Errorf("Status = %d, want 429", se.Status)
	}

	if se.Body != "too fast" {
		t.Errorf("Body = %q, want 'too fast'", se.Body)
	}

	if !errors.Is(se, apierr.ErrRateLimited) {
		t.Error("StatusError should also match via errors.Is")
	}
}

func TestStatusError_IsRetryable(t *testing.T) {
	t.Parallel()

	retryable := apierr.NewStatusError(429, "", apierr.ErrRateLimited)
	if !apierr.IsRetryable(retryable) {
		t.Error("StatusError wrapping ErrRateLimited should be retryable")
	}

	notRetryable := apierr.NewStatusError(404, "", apierr.ErrNotFound)
	if apierr.IsRetryable(notRetryable) {
		t.Error("StatusError wrapping ErrNotFound should not be retryable")
	}
}

func TestStatusError_DoubleWrapped(t *testing.T) {
	t.Parallel()

	se := apierr.NewStatusError(503, "maintenance", apierr.ErrServerError)
	vendorErr := fmt.Errorf("erp: %w", se)
	pipelineErr := fmt.Errorf("sync: %w", vendorErr)

	// errors.Is works through the chain.
	if !errors.Is(pipelineErr, apierr.ErrServerError) {
		t.Error("double-wrapped StatusError should match sentinel")
	}

	// errors.As works through the chain.
	var extracted *apierr.StatusError
	if !errors.As(pipelineErr, &extracted) {
		t.Fatal("double-wrapped StatusError should be extractable")
	}

	if extracted.Status != 503 {
		t.Errorf("Status = %d, want 503", extracted.Status)
	}

	// IsRetryable works through the chain.
	if !apierr.IsRetryable(pipelineErr) {
		t.Error("double-wrapped server error should be retryable")
	}
}

func TestFromStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code int
		want error
	}{
		{200, nil},
		{201, nil},
		{204, nil},
		{299, nil},
		{400, apierr.ErrInvalidInput},
		{401, apierr.ErrUnauthorized},
		{403, apierr.ErrForbidden},
		{404, apierr.ErrNotFound},
		{408, apierr.ErrTimeout},
		{409, apierr.ErrConflict},
		{429, apierr.ErrRateLimited},
		{422, apierr.ErrAPIError},
		{500, apierr.ErrServerError},
		{502, apierr.ErrServerError},
		{503, apierr.ErrServerError},
		{418, apierr.ErrUnexpectedStatus}, // teapot.
		{300, apierr.ErrUnexpectedStatus}, // redirect.
	}

	for _, tt := range tests {
		got := apierr.FromStatus(tt.code)
		if !errors.Is(got, tt.want) {
			t.Errorf("FromStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestFromGRPCCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code uint32
		want error
	}{
		{0, nil},                         // OK.
		{3, apierr.ErrInvalidInput},      // INVALID_ARGUMENT.
		{4, apierr.ErrTimeout},           // DEADLINE_EXCEEDED.
		{5, apierr.ErrNotFound},          // NOT_FOUND.
		{6, apierr.ErrConflict},          // ALREADY_EXISTS.
		{7, apierr.ErrForbidden},         // PERMISSION_DENIED.
		{8, apierr.ErrRateLimited},       // RESOURCE_EXHAUSTED.
		{13, apierr.ErrServerError},      // INTERNAL.
		{14, apierr.ErrServerError},      // UNAVAILABLE.
		{16, apierr.ErrUnauthorized},     // UNAUTHENTICATED.
		{99, apierr.ErrUnexpectedStatus}, // Unknown code.
	}

	for _, tt := range tests {
		got := apierr.FromGRPCCode(tt.code)
		if !errors.Is(got, tt.want) {
			t.Errorf("FromGRPCCode(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestRetryableStatus(t *testing.T) {
	t.Parallel()

	retryable := []int{408, 429, 500, 502, 503, 599}
	for _, code := range retryable {
		if !apierr.RetryableStatus(code) {
			t.Errorf("RetryableStatus(%d) = false, want true", code)
		}
	}

	permanent := []int{200, 204, 301, 400, 401, 403, 404, 409, 410, 422, 499}
	for _, code := range permanent {
		if apierr.RetryableStatus(code) {
			t.Errorf("RetryableStatus(%d) = true, want false", code)
		}
	}
}

// TestRetryableStatus_AgreesWithSentinelPath pins the documented equivalence:
// RetryableStatus(code) == IsRetryable(FromStatus(code)) for every HTTP code.
func TestRetryableStatus_AgreesWithSentinelPath(t *testing.T) {
	t.Parallel()

	for code := 100; code < 600; code++ {
		want := apierr.IsRetryable(apierr.FromStatus(code))
		if got := apierr.RetryableStatus(code); got != want {
			t.Errorf("RetryableStatus(%d) = %v, but IsRetryable(FromStatus(%d)) = %v", code, got, code, want)
		}
	}
}

func TestRetryableGRPCCode(t *testing.T) {
	t.Parallel()

	// DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED, ABORTED, INTERNAL, UNAVAILABLE.
	retryable := []uint32{4, 8, 10, 13, 14}
	for _, code := range retryable {
		if !apierr.RetryableGRPCCode(code) {
			t.Errorf("RetryableGRPCCode(%d) = false, want true", code)
		}
	}

	// OK, CANCELLED, UNKNOWN, INVALID_ARGUMENT, NOT_FOUND, ALREADY_EXISTS,
	// PERMISSION_DENIED, FAILED_PRECONDITION, OUT_OF_RANGE, UNIMPLEMENTED,
	// DATA_LOSS, UNAUTHENTICATED.
	permanent := []uint32{0, 1, 2, 3, 5, 6, 7, 9, 11, 12, 15, 16}
	for _, code := range permanent {
		if apierr.RetryableGRPCCode(code) {
			t.Errorf("RetryableGRPCCode(%d) = true, want false", code)
		}
	}
}

// TestFromGRPCCode_Aborted pins the ABORTED → ErrConflict mapping (the gRPC
// analogue of HTTP 409), which is deliberately non-retryable as a sentinel
// while RetryableGRPCCode(10) is true — see the RetryableGRPCCode docs.
func TestFromGRPCCode_Aborted(t *testing.T) {
	t.Parallel()

	if err := apierr.FromGRPCCode(10); !errors.Is(err, apierr.ErrConflict) {
		t.Fatalf("FromGRPCCode(10) = %v, want ErrConflict", err)
	}
}
