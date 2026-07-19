package web_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/duizendstra/alexandria/go/platform/apierr"
	"github.com/duizendstra/alexandria/go/platform/web"
)

func ExampleEncodeJSON() {
	w := httptest.NewRecorder()

	web.EncodeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	fmt.Println(w.Code)
	fmt.Print(w.Body.String())
	// Output:
	// 200
	// {"status":"ok"}
}

func ExampleDecodeJSON() {
	type createUserRequest struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader(`{"name":"Ada"}`))
	w := httptest.NewRecorder()

	// 0 applies the DefaultMaxJSONSize limit (1MB).
	body, err := web.DecodeJSON[createUserRequest](w, req, 0)
	if err != nil {
		web.WriteError(w, err)

		return
	}

	fmt.Println(body.Name)
	// Output: Ada
}

func ExampleWriteError() {
	w := httptest.NewRecorder()

	// apierr sentinels map to their canonical HTTP status codes.
	web.WriteError(w, apierr.ErrNotFound)

	fmt.Println(w.Code)
	fmt.Print(w.Body.String())
	// Output:
	// 404
	// {"error":"not found"}
}

func ExampleContentTypeJSONMiddleware() {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := web.ContentTypeJSONMiddleware(next)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Print(w.Body.String())
	// Output:
	// 415
	// {"error":"Content-Type must be application/json"}
}

func ExampleNewRequestWithContext() {
	ctx := context.Background()

	req, err := web.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/health", http.NoBody)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(req.Method, req.URL.String())
	// Output: GET https://api.example.com/health
}
