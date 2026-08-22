package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/duizendstra/alexandria/go/platform/apierr"
	"github.com/duizendstra/alexandria/go/retry"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/status"
)

const (
	// defaultMaxAttempts is the default maximum execution attempts.
	defaultMaxAttempts = 10
	defaultBaseDelay   = 100 * time.Millisecond
	defaultMaxCap      = 5 * time.Second
	maxAttemptShift    = 30
	jitterDivisor      = 5
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

// Option configures behavior for WithRetry and WithRetryVal.
type Option func(*config)

type config struct {
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	onRetry        func(attempt int, delay time.Duration, err error)
}

// WithMaxAttempts configures the maximum attempts.
func WithMaxAttempts(attempts int) Option {
	return func(c *config) {
		if attempts > 0 {
			c.maxAttempts = attempts
		}
	}
}

// WithInitialBackoff configures the initial base delay for exponential backoff (e.g. 500ms).
func WithInitialBackoff(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.initialBackoff = d
		}
	}
}

// WithMaxBackoff configures the maximum backoff duration ceiling (e.g. 32s for Drive batch operations).
func WithMaxBackoff(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.maxBackoff = d
		}
	}
}

// WithOnRetry registers an observability callback invoked before each retry sleep.
func WithOnRetry(fn func(attempt int, delay time.Duration, err error)) Option {
	return func(c *config) {
		c.onRetry = fn
	}
}

// WithRetry executes an operation callback function with exponential backoff and GCP-specific error classification.
// It fails fast on permanent failures (like OAuth/impersonation issues) and retries on transient errors.
func WithRetry(ctx context.Context, operation func() error, opts ...Option) error {
	_, err := WithRetryVal(ctx, func() (struct{}, error) {
		return struct{}{}, operation()
	}, opts...)

	return err
}

// WithRetryVal executes an operation callback function with exponential backoff,
// GCP-specific error classification, Retry-After adherence, and returns the result.
func WithRetryVal[T any](ctx context.Context, operation func() (T, error), opts ...Option) (T, error) {
	var zero T
	cfg := config{
		maxAttempts: defaultMaxAttempts,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	attempt := 0
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return zero, fmt.Errorf("gcp operation failed: %w", err)
		}

		attempt++
		val, opErr := operation()
		if opErr == nil {
			return val, nil
		}

		classifiedErr := Classify(ctx, opErr, attempt)
		if retry.IsPermanent(classifiedErr) {
			return zero, fmt.Errorf("gcp operation failed after %d attempts: %w", attempt, classifiedErr)
		}

		if attempt >= cfg.maxAttempts {
			return zero, fmt.Errorf("gcp operation failed after %d attempts: %w", attempt, classifiedErr)
		}

		delay := calculateDelay(classifiedErr, attempt-1, &cfg, time.Now())
		if cfg.onRetry != nil {
			cfg.onRetry(attempt, delay, classifiedErr)
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(delay)
		}

		select {
		case <-timer.C:
		case <-ctx.Done():
			return zero, fmt.Errorf("gcp operation failed: %w", ctx.Err())
		}
	}
}

func calculateDelay(err error, attempt int, cfg *config, now time.Time) time.Duration {
	var delay time.Duration
	if cfg.initialBackoff > 0 || cfg.maxBackoff > 0 {
		base := cfg.initialBackoff
		if base <= 0 {
			base = defaultBaseDelay
		}
		maxCap := cfg.maxBackoff
		if maxCap <= 0 {
			maxCap = defaultMaxCap
		}
		shift := min(max(attempt, 0), maxAttemptShift)
		b := min(time.Duration(int64(1)<<uint(shift))*base, maxCap)
		jit := time.Duration(0)
		if b/jitterDivisor > 0 {
			jit = time.Duration(rand.Int64N(int64(b / jitterDivisor))) // #nosec G404
		}
		delay = b + jit
	} else {
		delay = retry.Backoff(attempt)
	}

	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok && apiErr.Header != nil {
		if headerDelay, ok := parseRetryAfter(apiErr.Header.Get("Retry-After"), now); ok {
			delay = max(delay, headerDelay)
			if cfg.maxBackoff > 0 {
				delay = min(delay, cfg.maxBackoff)
			}
		}
	}

	return delay
}

func parseRetryAfter(header string, now time.Time) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	if secs, err := strconv.Atoi(header); err == nil {
		if secs < 0 {
			return 0, false
		}

		return time.Duration(secs) * time.Second, true
	}

	if date, err := http.ParseTime(header); err == nil {
		return max(date.Sub(now), 0), true
	}

	return 0, false
}

// Classify determines whether an error should be retried.
// It returns a permanent error (wrapped via retry.Permanent) for permanent failures, or the original error to allow retrying.
//
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
	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok {
		return classifyAPIError(apiErr, attempt)
	}

	// 3. gRPC Status Check (Addresses the gRPC Error Blind Spot).
	if s, ok := status.FromError(err); ok {
		return classifyGRPCError(s, attempt)
	}

	// 4. Structured OAuth2 token endpoint check (RFC 6749). Preferred over
	// the string heuristics below because it keys on the status code and
	// error code the endpoint actually returned, surviving SDK rewordings.
	if oauthErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		return classifyOAuthRetrieveError(err, oauthErr, attempt)
	}

	// 5. Typed Network Check.
	if _, ok := errors.AsType[net.Error](err); ok {
		logger().Warn("Transient network error, will retry",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()))

		return err
	}

	// 6. Explicitly retry unexpected network disconnections (UnexpectedEOF).
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

	// 7. Everything typed has been ruled out: fall back to string heuristics.
	return classifyByErrorString(err, attempt)
}

// classifyOAuthRetrieveError classifies a structured *oauth2.RetrieveError
// from a token endpoint using its HTTP status and RFC 6749 error code
// instead of error-message text.
func classifyOAuthRetrieveError(err error, rErr *oauth2.RetrieveError, attempt int) error {
	statusCode := 0
	if rErr.Response != nil {
		statusCode = rErr.Response.StatusCode
	}

	// An explicit RFC 6749 error code names an authorization or client
	// configuration problem (e.g. DWD misconfiguration) no retry can fix.
	if isPermanentOAuthErrorCode(rErr.ErrorCode) {
		logger().Error("Permanent OAuth2 token endpoint error, not retrying",
			slog.Int("attempt", attempt),
			slog.Int("http_code", statusCode),
			slog.String("oauth_error_code", rErr.ErrorCode),
			slog.String("error", err.Error()))

		//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
		return retry.Permanent(err)
	}

	// Server-side failures and throttling at the token endpoint are transient.
	if apierr.RetryableStatus(statusCode) {
		logger().Warn("Transient OAuth2 token endpoint error, will retry",
			slog.Int("attempt", attempt),
			slog.Int("http_code", statusCode),
			slog.String("oauth_error_code", rErr.ErrorCode),
			slog.String("error", err.Error()))

		return err
	}

	// Remaining 4xx (or unknown) responses reject the request itself.
	logger().Error("Permanent OAuth2 token endpoint error, not retrying",
		slog.Int("attempt", attempt),
		slog.Int("http_code", statusCode),
		slog.String("oauth_error_code", rErr.ErrorCode),
		slog.String("error", err.Error()))

	//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
	return retry.Permanent(err)
}

// isPermanentOAuthErrorCode reports whether code is an RFC 6749 §5.2 token
// endpoint error code that indicates a permanent authorization or client
// configuration failure.
func isPermanentOAuthErrorCode(code string) bool {
	switch code {
	case "invalid_request",
		"invalid_client",
		"invalid_grant",
		"unauthorized_client",
		"unsupported_grant_type",
		"invalid_scope":
		return true
	default:
		return false
	}
}

// classifyByErrorString is the last-resort cold path: substring matching on
// error text, pinned to strings observed in the Google SDKs (impersonation
// helpers and oauth2 token fetches that flatten their cause into a message).
// It only runs after every structured check in Classify has failed to match,
// because upstream wording changes can silently flip these heuristics —
// prefer adding a typed check over extending this list.
func classifyByErrorString(err error, attempt int) error {
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
// HTTP retryability (408/429/5xx) is delegated to apierr.RetryableStatus —
// the ecosystem's single classification table — with one GCP-specific
// extension: 403 responses whose reason is quota/rate-limiting.
func classifyAPIError(apiErr *googleapi.Error, attempt int) error {
	isRetryable := false
	var logMsg string

	if apierr.RetryableStatus(apiErr.Code) {
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

// classifyGRPCError maps modern cloud client gRPC status codes to
// retryability profiles. The transient set (DEADLINE_EXCEEDED,
// RESOURCE_EXHAUSTED, ABORTED, INTERNAL, UNAVAILABLE) is delegated to
// apierr.RetryableGRPCCode — the ecosystem's single classification table.
func classifyGRPCError(s *status.Status, attempt int) error {
	if apierr.RetryableGRPCCode(uint32(s.Code())) {
		logger().Warn("Transient gRPC error, will retry",
			slog.Int("attempt", attempt),
			slog.String("grpc_code", s.Code().String()),
			slog.String("error", s.Message()))

		//nolint:wrapcheck // Returning raw error from status evaluates properly at domain layer.
		return s.Err()
	}

	logger().Error("Permanent gRPC error, not retrying",
		slog.Int("attempt", attempt),
		slog.String("grpc_code", s.Code().String()),
		slog.String("error", s.Message()))

	//nolint:wrapcheck // retry.Permanent wraps errors internally to mark them as permanent for the retry runner.
	return retry.Permanent(s.Err())
}
