// Package web provides general technical HTTP server, client, and response utilities
// designed as project-agnostic components.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/duizendstra/alexandria/go/platform/apierr"
)

// DefaultMaxJSONSize is the default limit for incoming JSON bodies (1MB).
const DefaultMaxJSONSize = 1024 * 1024

// Sentinel errors for the web package to satisfy static analysis and enable robust matching.
var (
	ErrTrailingGarbage = errors.New("request body must only contain a single JSON object")
	ErrPanicRecovered  = errors.New("API recovered from panic")
)

// EncodeJSON marshals v to JSON and writes it to w with the given status code.
func EncodeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", slog.Any("err", err))
	}
}

// DecodeJSON reads r.Body up to maxSize bytes and decodes it into a value of type T.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request, maxSize int64) (T, error) {
	var v T
	limit := maxSize
	if limit <= 0 {
		limit = DefaultMaxJSONSize
	}

	// Limit reader to shield against memory exhaustion attacks.
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("failed to decode JSON body: %w", err)
	}

	// Ensure there is no trailing garbage in the body.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return v, ErrTrailingGarbage
	}

	return v, nil
}

// Statuses WriteError honours from an [apierr.StatusError]: the 3xx, 4xx and
// 5xx range, passed through as-is. Anything else — 0, 1xx, 2xx, or an
// out-of-range upstream code — degrades to the generic 500:
// [http.ResponseWriter.WriteHeader] panics outside 100–999, and a JSON error
// body on an informational or success status is meaningless.
const (
	minHonouredStatus = http.StatusMultipleChoices
	maxHonouredStatus = 599
)

// WriteError writes a standardized JSON error response mapped from the given error.
//
// An [apierr.StatusError] sets the response status only when its Status is a
// 3xx, 4xx or 5xx code (300–599), written as-is; any other Status falls back
// to 500 instead of panicking in WriteHeader. The client sees the sentinel and
// status alone — the upstream response excerpt in StatusError.Body never
// reaches the response, so log the full error server-side before calling
// WriteError.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	msg := "An internal server error occurred"

	if se, ok := errors.AsType[*apierr.StatusError](err); ok {
		if isHonouredStatus(se.Status) {
			status = se.Status
			// apierr's own status-only wording, without the body.
			msg = (&apierr.StatusError{Status: se.Status, Err: se.Err}).Error()
		}
	} else {
		// Map standard sentinels.
		switch {
		case errors.Is(err, apierr.ErrInvalidInput):
			status = http.StatusBadRequest
			msg = err.Error()
		case errors.Is(err, apierr.ErrUnauthorized):
			status = http.StatusUnauthorized
			msg = err.Error()
		case errors.Is(err, apierr.ErrForbidden):
			status = http.StatusForbidden
			msg = err.Error()
		case errors.Is(err, apierr.ErrNotFound):
			status = http.StatusNotFound
			msg = err.Error()
		case errors.Is(err, apierr.ErrTimeout):
			status = http.StatusRequestTimeout
			msg = err.Error()
		case errors.Is(err, apierr.ErrConflict):
			status = http.StatusConflict
			msg = err.Error()
		case errors.Is(err, apierr.ErrRateLimited):
			status = http.StatusTooManyRequests
			msg = err.Error()
		}
	}

	EncodeJSON(w, status, map[string]string{"error": msg})
}

// isHonouredStatus reports whether status is a 3xx, 4xx or 5xx code.
func isHonouredStatus(status int) bool {
	return status >= minHonouredStatus && status <= maxHonouredStatus
}

// RecoveryMiddleware catches panics, logs them, and returns a 500 status.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				var err error
				if recErr, ok := rec.(error); ok {
					err = recErr
				} else {
					err = fmt.Errorf("%w: %v", ErrPanicRecovered, rec)
				}

				slog.Error("API recovered from panic",
					slog.Any("err", err),
					slog.String("stack", string(debug.Stack())),
					slog.String("path", r.URL.Path))

				WriteError(w, apierr.ErrServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ContentTypeJSONMiddleware ensures that requests to POST/PUT/PATCH contain JSON content-type.
func ContentTypeJSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
				EncodeJSON(w, http.StatusUnsupportedMediaType, map[string]string{
					"error": "Content-Type must be application/json",
				})

				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// NewRequestWithContext is a helper to satisfy static context analysis for HTTP client calls.
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	return req, nil
}
