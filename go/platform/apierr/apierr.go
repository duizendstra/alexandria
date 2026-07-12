package apierr

import (
	"errors"
	"fmt"
)

// HTTP status sentinels. Names are semantic, not HTTP-specific — they
// map cleanly to both HTTP status codes and gRPC status codes.
//
//	| Sentinel           | HTTP | gRPC               |
//	|--------------------|------|---------------------|
//	| ErrInvalidInput    | 400  | INVALID_ARGUMENT    |
//	| ErrUnauthorized    | 401  | UNAUTHENTICATED     |
//	| ErrForbidden       | 403  | PERMISSION_DENIED   |
//	| ErrNotFound        | 404  | NOT_FOUND           |
//	| ErrTimeout         | 408  | DEADLINE_EXCEEDED   |
//	| ErrConflict        | 409  | ALREADY_EXISTS      |
//	| ErrRateLimited     | 429  | RESOURCE_EXHAUSTED  |
//	| ErrServerError     | 5xx  | INTERNAL            |
//	| ErrUnexpectedStatus| other| UNKNOWN             |
var (
	// ErrInvalidInput is returned when the request is malformed or fails
	// client-side validation (400 / INVALID_ARGUMENT). This covers
	// pre-flight checks like invalid license plate format or empty address.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when credentials are rejected (401 / UNAUTHENTICATED).
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the caller lacks permissions (403 / PERMISSION_DENIED).
	ErrForbidden = errors.New("forbidden")

	// ErrNotFound is returned when the resource does not exist (404 / NOT_FOUND).
	ErrNotFound = errors.New("not found")

	// ErrTimeout is returned when the request times out (408 / DEADLINE_EXCEEDED).
	ErrTimeout = errors.New("timeout")

	// ErrConflict is returned on duplicate or conflicting state (409 / ALREADY_EXISTS).
	ErrConflict = errors.New("conflict")

	// ErrRateLimited is returned when the API throttles requests (429 / RESOURCE_EXHAUSTED).
	ErrRateLimited = errors.New("rate limited")

	// ErrServerError is returned for server-side failures (5xx / INTERNAL).
	ErrServerError = errors.New("server error")

	// ErrUnexpectedStatus is returned for any unhandled status (UNKNOWN).
	ErrUnexpectedStatus = errors.New("unexpected status")
)

// Client behavior sentinels.
var (
	// ErrRetriesExceeded is returned when all retry attempts are exhausted.
	ErrRetriesExceeded = errors.New("retries exceeded")

	// ErrAuthFailed is returned when authentication setup fails
	// (e.g., token provider error, not an HTTP 401).
	ErrAuthFailed = errors.New("authentication failed")

	// ErrAPIError is returned when the vendor API indicates a logical
	// failure in its response envelope (e.g., success=false, GraphQL errors).
	ErrAPIError = errors.New("API error")
)

// IsRetryable reports whether err is a transient failure that may
// succeed on retry. Consumers should apply backoff before retrying.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrServerError) ||
		errors.Is(err, ErrTimeout)
}

// StatusError wraps a sentinel error with the HTTP status code and
// response body excerpt. Use [errors.As] to extract context:
//
//	var se *apierr.StatusError
//	if errors.As(err, &se) {
//	    log.Info("failed", "status", se.Status, "body", se.Body)
//	}
type StatusError struct {
	// Status is the HTTP status code (e.g., 429).
	Status int
	// Body is a truncated excerpt of the response body (max 4096 bytes).
	Body string
	// Err is the sentinel error (e.g., ErrRateLimited).
	Err error
}

// NewStatusError creates a [StatusError] wrapping the given sentinel.
func NewStatusError(status int, body string, sentinel error) *StatusError {
	return &StatusError{Status: status, Body: body, Err: sentinel}
}

// Error returns a human-readable message including status and sentinel.
func (e *StatusError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s: %d %s", e.Err, e.Status, e.Body)
	}

	return fmt.Sprintf("%s: %d", e.Err, e.Status)
}

// Unwrap returns the sentinel error for [errors.Is] matching.
func (e *StatusError) Unwrap() error {
	return e.Err
}

// FromStatus maps a numeric status code to the corresponding sentinel error.
// Returns nil for success codes (2xx). This is protocol-agnostic — the
// status code can come from HTTP, gRPC, or any system that uses numeric codes.
//
//	if err := apierr.FromStatus(resp.StatusCode); err != nil {
//	    return err
//	}
//
// Status codes used by [FromStatus]. Defined here to avoid importing
// net/http — keeping this package dependency-free.
const (
	statusOK                  = 200
	statusMultipleChoices     = 300
	statusBadRequest          = 400
	statusUnauthorized        = 401
	statusForbidden           = 403
	statusNotFound            = 404
	statusTimeout             = 408
	statusConflict            = 409
	statusUnprocessableEntity = 422
	statusRateLimited         = 429
	statusInternalServerError = 500
)

func FromStatus(code int) error {
	switch {
	case code >= statusOK && code < statusMultipleChoices:
		return nil
	case code == statusBadRequest:
		return ErrInvalidInput
	case code == statusUnauthorized:
		return ErrUnauthorized
	case code == statusForbidden:
		return ErrForbidden
	case code == statusNotFound:
		return ErrNotFound
	case code == statusTimeout:
		return ErrTimeout
	case code == statusConflict:
		return ErrConflict
	case code == statusRateLimited:
		return ErrRateLimited
	case code == statusUnprocessableEntity:
		return ErrAPIError
	case code >= statusInternalServerError:
		return ErrServerError
	default:
		return ErrUnexpectedStatus
	}
}

// FromGRPCCode maps a gRPC status code to the corresponding sentinel error.
// Returns nil for OK. This enables gRPC-based vendor clients to use the
// same error contract as HTTP-based ones.
//
// gRPC codes are defined in google.golang.org/grpc/codes but we accept
// uint32 to avoid importing grpc in this dependency-free package.
func FromGRPCCode(code uint32) error {
	switch code {
	case grpcOK:
		return nil
	case grpcInvalidArgument:
		return ErrInvalidInput
	case grpcNotFound:
		return ErrNotFound
	case grpcAlreadyExists:
		return ErrConflict
	case grpcPermissionDenied:
		return ErrForbidden
	case grpcResourceExhausted:
		return ErrRateLimited
	case grpcUnauthenticated:
		return ErrUnauthorized
	case grpcUnavailable:
		return ErrServerError
	case grpcDeadlineExceeded:
		return ErrTimeout
	case grpcInternal:
		return ErrServerError
	default:
		return ErrUnexpectedStatus
	}
}

// gRPC status codes — mirrors google.golang.org/grpc/codes without the import.
const (
	grpcOK                = 0
	grpcInvalidArgument   = 3
	grpcNotFound          = 5
	grpcAlreadyExists     = 6
	grpcPermissionDenied  = 7
	grpcResourceExhausted = 8
	grpcUnauthenticated   = 16
	grpcUnavailable       = 14
	grpcDeadlineExceeded  = 4
	grpcInternal          = 13
)
