package web

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"foo": "bar"}

	EncodeJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"foo":"bar"}`, w.Body.String())
}

func TestDecodeJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("valid payload", func(t *testing.T) {
		r := httptest.NewRequestWithContext(ctx, "POST", "/", strings.NewReader(`{"foo":"bar"}`))
		w := httptest.NewRecorder()

		val, err := DecodeJSON[map[string]string](w, r, 0)
		require.NoError(t, err)
		assert.Equal(t, "bar", val["foo"])
	})

	t.Run("oversized payload", func(t *testing.T) {
		r := httptest.NewRequestWithContext(ctx, "POST", "/", strings.NewReader(`{"foo":"bar"}`))
		w := httptest.NewRecorder()

		_, err := DecodeJSON[map[string]string](w, r, 5) // Limit size to 5 bytes.
		assert.Error(t, err)
	})

	t.Run("trailing garbage", func(t *testing.T) {
		r := httptest.NewRequestWithContext(ctx, "POST", "/", strings.NewReader(`{"foo":"bar"}{} trailing`))
		w := httptest.NewRecorder()

		_, err := DecodeJSON[map[string]string](w, r, 0)
		assert.Error(t, err)
	})
}

func TestWriteError(t *testing.T) {
	t.Run("standard error", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, errors.New("something went wrong"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"An internal server error occurred"}`, w.Body.String())
	})

	t.Run("apierr sentinel input", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, apierr.ErrInvalidInput)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"error":"invalid input"}`, w.Body.String())
	})

	t.Run("apierr sentinel not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, apierr.ErrNotFound)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.JSONEq(t, `{"error":"not found"}`, w.Body.String())
	})

	t.Run("StatusError formatting", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, apierr.NewStatusError(http.StatusConflict, "already exists", apierr.ErrConflict))

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.JSONEq(t, `{"error":"conflict: 409 already exists"}`, w.Body.String())
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	ctx := context.Background()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic error")
	})

	mw := RecoveryMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(ctx, "GET", "/", http.NoBody)

	assert.NotPanics(t, func() {
		mw.ServeHTTP(w, r)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"An internal server error occurred"}`, w.Body.String())
}

func TestContentTypeJSONMiddleware(t *testing.T) {
	ctx := context.Background()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := ContentTypeJSONMiddleware(handler)

	t.Run("GET request bypasses check", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, "GET", "/", http.NoBody)
		mw.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST request without json content type fails", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader([]byte(`{"foo":"bar"}`)))
		r.Header.Set("Content-Type", "text/plain")
		mw.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	})

	t.Run("POST request with json content type passes", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader([]byte(`{"foo":"bar"}`)))
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
		mw.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
