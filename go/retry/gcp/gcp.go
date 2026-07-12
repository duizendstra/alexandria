package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/duizendstra/alexandria/go/retry"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// defaultMaxAttempts is the default maximum execution attempts.
	defaultMaxAttempts = 10
)

//nolint:gochecknoglobals // defaultLogger allows configuring package-level logging.
var defaultLogger atomic.Pointer[slog.Logger]

// SetLogger sets the logger to be used by the package.
// If nil is passed, it will revert to using slog.Default().
func SetLogger(l *slog.Logger) {
	if l == nil {
		defaultLogger.Store(nil)

		return
	}
	defaultLogger.Store(l)
}

func logger() *slog.Logger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}

	return slog.Default()
}

// Option configures behavior for WithRetry.
type Option func(*config)

type config struct {
	maxAttempts int
}

// WithMaxAttempts configures the maximum attempts.
func WithMaxAttempts(attempts int) Option {
	return func(c *config) {
		if attempts > 0 {
			c.maxAttempts = attempts
		}
	}
}

// WithRetry executes an operation callback function with exponential backoff and GCP-specific error classification.
// It fails fast on permanent failures (like OAuth/impersonation issues) and retries on transient errors.
func WithRetry(ctx context.Context, operation func() error, opts ...Option) error {
	cfg := config{maxAttempts: defaultMaxAttempts}
	for _, opt := range opts {
		opt(&cfg)
	}

	attempt := 0
	op := func() error {
		if err := ctx.Err(); err != nil {
			return retry.Permanent(ctx.Err())
		}

		attempt++
		err := operation()
		if err == nil {
			return nil
		}

		return Classify(ctx, err, attempt)
	}

	if err := retry.Do(ctx, cfg.maxAttempts, op); err != nil {
		return fmt.Errorf("gcp operation failed after %d attempts: %w", attempt, err)
	}

	return nil
}

// Classify determines whether an error should be retried.
// It returns a permanent error (wrapped via retry.Permanent) for permanent failures, or the original error to allow retrying.
//
//nolint:cyclop // Classify is a flat, switch-like classification dispatcher where higher complexity is expected and highly readable.
func Classify(ctx context.Context, err error, attempt int) error {
	if err == nil {
		return nil
	}

	// 1. Only abort if the parent context is done. Sub-timeouts are transient!
	if ctx.Err() != nil {
		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(ctx.Err())
	}
	if errors.Is(err, context.Canceled) {
		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	// 2. Typed API Check.
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return classifyAPIError(apiErr, attempt)
	}

	// 3. gRPC Status Check (Addresses the gRPC Error Blind Spot).
	if s, ok := status.FromError(err); ok {
		return classifyGRPCError(s, attempt)
	}

	// 4. Typed Network Check.
	var netErr net.Error
	if errors.As(err, &netErr) {
		logger().Warn("Transient network error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}

	// 5. Explicitly retry unexpected network disconnections (UnexpectedEOF).
	if errors.Is(err, io.EOF) {
		logger().Warn("Transient end-of-file error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		logger().Warn("Transient unexpected EOF error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}

	// 6. Cold-path: string matching fallback. Ensures transient network errors
	// wrapped inside oauth2 token fetches are not incorrectly flagged permanent.
	errStr := err.Error()
	if strings.Contains(errStr, "impersonate:") ||
		strings.Contains(errStr, "unauthorized_client") ||
		strings.Contains(errStr, "invalid_grant") {
		logger().Error("Permanent OAuth2/DWD error, not retrying",
			slog.Int("attempt", attempt),
			slog.String("error", errStr))

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	// Catch transient timeouts wrapped in oauth token strings.
	if strings.Contains(errStr, "oauth2: cannot fetch token") {
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connection reset") {
			logger().Warn("Transient OAuth network failure, will retry",
				slog.Int("attempt", attempt),
				slog.String("error", errStr))

			return err
		}

		logger().Error("Permanent OAuth2 validation error, not retrying",
			slog.Int("attempt", attempt),
			slog.String("error", errStr))

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	// Non-API errors (OAuth failures, scope mismatches, other errors) are permanent.
	logger().Error("Non-API error, not retrying",
		slog.Int("attempt", attempt),
		slog.String("error", errStr))

	//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
	return retry.Permanent(err)
}

// classifyAPIError determines whether a googleapi.Error is retryable.
func classifyAPIError(apiErr *googleapi.Error, attempt int) error {
	isRetryable := false
	var logMsg string

	if apiErr.Code == http.StatusTooManyRequests || apiErr.Code >= 500 {
		isRetryable = true
		logMsg = "Retryable API error, will retry"
	} else if apiErr.Code == http.StatusForbidden {
		for _, item := range apiErr.Errors {
			if item.Reason == "rateLimitExceeded" ||
				item.Reason == "userRateLimitExceeded" ||
				item.Reason == "quotaExceeded" {
				isRetryable = true
				logMsg = "Retryable API quota/rate-limit error (403), will retry"

				break
			}
		}
	}

	if isRetryable {
		logger().Warn(logMsg,
			slog.Int("attempt", attempt),
			slog.Int("http_code", apiErr.Code),
			slog.String("error", apiErr.Message))

		return apiErr
	}

	// Permanent API error (e.g., non-retryable 403 Forbidden, 404 Not Found).
	logger().Error("Permanent API error, not retrying",
		slog.Int("attempt", attempt),
		slog.Int("http_code", apiErr.Code),
		slog.String("error", apiErr.Message))

	//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
	return retry.Permanent(apiErr)
}

// classifyGRPCError maps modern cloud client gRPC status codes to retryability profiles.
//
//nolint:exhaustive // Only subset of standard retryable codes are explicitly switched on, default handles non-retryable codes.
func classifyGRPCError(s *status.Status, attempt int) error {
	switch s.Code() {
	case codes.ResourceExhausted, // HTTP 429.
		codes.Unavailable,     // Connection drops.
		codes.Internal,        // Remote exceptions.
		codes.Aborted,         // Concurrent transaction interruptions.
		codes.DeadlineExceeded: // Request Timeout.
		logger().Warn("Transient gRPC error, will retry",
			slog.Int("attempt", attempt),
			slog.String("grpc_code", s.Code().String()),
			slog.String("error", s.Message()))

		//nolint:wrapcheck // Returning raw error from status evaluates properly at domain layer.
		return s.Err()

	default:
		logger().Error("Permanent gRPC error, not retrying",
			slog.Int("attempt", attempt),
			slog.String("grpc_code", s.Code().String()),
			slog.String("error", s.Message()))

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(s.Err())
	}
}
