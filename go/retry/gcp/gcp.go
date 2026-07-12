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

// WithRetry executes an operation callback function with exponential backoff and GCP-specific error classification.
// It fails fast on permanent failures (like OAuth/impersonation issues) and retries on transient errors.
func WithRetry(ctx context.Context, operation func() error) error {
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

	// Using a default of 10 maximum attempts, consistent with standard retry windows.
	const maxAttempts = 10
	if err := retry.Do(ctx, maxAttempts, op); err != nil {
		return fmt.Errorf("gcp operation failed after %d attempts: %w", attempt, err)
	}

	return nil
}

// Classify determines whether an error should be retried.
// It returns a permanent error (wrapped via retry.Permanent) for permanent failures, or the original error to allow retrying.
func Classify(ctx context.Context, err error, attempt int) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		if ctx.Err() != nil {
			//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
			return retry.Permanent(ctx.Err())
		}

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	// Check for permanent OAuth2 / impersonation / DWD authorization errors before standard network classification.
	errStr := err.Error()
	if strings.Contains(errStr, "impersonate:") ||
		strings.Contains(errStr, "unauthorized_client") ||
		strings.Contains(errStr, "invalid_grant") ||
		strings.Contains(errStr, "oauth2: cannot fetch token") {
		logger().Error("Permanent OAuth2/DWD error, not retrying",
			slog.Int("attempt", attempt),
			slog.String("error", errStr))

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return classifyAPIError(apiErr, attempt)
	}

	// Detect if the error implements standard Go net.Error interface or is io.EOF.
	var netErr net.Error
	if errors.As(err, &netErr) {
		logger().Warn("Transient network error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}

	if errors.Is(err, io.EOF) {
		logger().Warn("Transient end-of-file error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}

	// Non-API errors (OAuth failures, scope mismatches, other errors) are permanent.
	logger().Error("Non-API error, not retrying",
		slog.Int("attempt", attempt),
		slog.String("error", err.Error()))

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
