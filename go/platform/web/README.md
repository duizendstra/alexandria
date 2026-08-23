# go/platform/web

`go/platform/web` provides project-agnostic HTTP server, client, and response utilities: JSON encoding and bounded decoding, a single error-to-response mapper built on `go/platform/apierr`, and two middlewares.

## Features

- **One Place That Turns An Error Into A Response**: `WriteError` maps an `apierr` sentinel or an `*apierr.StatusError` onto a status code and a `{"error": "..."}` body. Anything it does not recognise becomes a generic 500 — an unexpected error cannot narrate itself to the client.
- **Bounded JSON Decoding**: `DecodeJSON[T]` wraps the body in `http.MaxBytesReader`, rejects unknown fields, and rejects anything trailing the first JSON value (`ErrTrailingGarbage`). A `maxSize` of zero or less applies `DefaultMaxJSONSize` (1 MB).
- **Panic Recovery**: `RecoveryMiddleware` converts a panicking handler into a logged 500 with a stack trace, rather than a dropped connection.
- **Content-Type Gate**: `ContentTypeJSONMiddleware` answers 415 to a `POST`, `PUT`, or `PATCH` whose `Content-Type` is present and not `application/json`. Parameters such as `; charset=utf-8` are accepted; a request with **no** `Content-Type` header is passed through.
- **Context-Carrying Client Requests**: `NewRequestWithContext` wraps `http.NewRequestWithContext` so client call sites satisfy context linters without hand-written error wrapping.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/web
```

## Quick Start

### A JSON Handler That Maps Its Own Failures

```go
package main

import (
	"log/slog"
	"net/http"

	"github.com/duizendstra/alexandria/go/platform/apierr"
	"github.com/duizendstra/alexandria/go/platform/web"
)

type createUserRequest struct {
	Name string `json:"name"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	// 0 applies DefaultMaxJSONSize. Unknown fields and trailing content are
	// rejected, so a malformed body never reaches the domain.
	req, err := web.DecodeJSON[createUserRequest](w, r, 0)
	if err != nil {
		web.WriteError(w, apierr.ErrInvalidInput)

		return
	}

	if req.Name == "" {
		web.WriteError(w, apierr.ErrInvalidInput) // 400
		return
	}

	// A sentinel from any layer maps to its canonical status. Log the full
	// error server-side first: the client is shown the sentinel alone.
	if err := store(req.Name); err != nil {
		slog.Error("create user", slog.Any("err", err))
		web.WriteError(w, err)

		return
	}

	web.EncodeJSON(w, http.StatusCreated, map[string]string{"name": req.Name})
}

func store(string) error { return nil }

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", createUser)

	// Outermost middleware recovers panics; the gate rejects non-JSON bodies.
	handler := web.RecoveryMiddleware(web.ContentTypeJSONMiddleware(mux))

	srv := &http.Server{Addr: ":8080", Handler: handler}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", slog.Any("err", err))
	}
}
```

## SRE & Performance Hardening details

1. **Memory Exhaustion Bounds**: every decode runs through `http.MaxBytesReader`, so a hostile or broken client cannot make a handler buffer an unbounded body. The limit is per-call, and a caller that passes nothing still gets the 1 MB default rather than no limit.
2. **Upstream Detail Stays Server-Side**: `apierr.StatusError` carries an excerpt of the upstream response body for diagnosis. `WriteError` writes the sentinel and the status only — the excerpt never reaches the client, so a proxy's error page or a vendor's message cannot be reflected out of the service.
3. **`WriteHeader` Is Never Handed An Invalid Status**: `net/http` panics when written a status outside 100–999. `WriteError` honours 300–599 and degrades everything else to 500, so an upstream code copied into a `StatusError` cannot turn into a panic inside the response path.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Small JSON HTTP services**: handlers that decode a request, call a domain,
  and need one consistent failure shape without a framework.
- **Adapters fronting a vendor API**: code that receives an upstream status and
  must decide what — if anything — of it the caller is allowed to see.

### Load-Bearing Promises
1. **An Unrecognised Error Never Speaks To The Client**: an error that is neither
   an `apierr` sentinel nor an `*apierr.StatusError` produces a 500 with a fixed
   generic message. Its own text is never written to the response.
2. **A Recognised Sentinel Sets Its Canonical Status**: an `apierr` sentinel
   resolves to its HTTP status and its own wording rather than the generic 500.
3. **A `StatusError` Body Stays Server-Side**: the upstream excerpt in
   `StatusError.Body` never appears in the response — at a passed-through
   status, at a degraded one, or at a rejected one.
4. **A Status Cannot Panic The Response Path**: every 3xx, 4xx and 5xx code is
   passed through as-is; anything else — negative, zero, 1xx, 2xx, or above 599
   — degrades to 500 instead of reaching `WriteHeader`.
5. **A Decode Is Bounded And Whole**: a body over the limit fails, and content
   trailing the first JSON value fails. A handler cannot silently act on the
   first object of a body that carried two.
6. **A Panicking Handler Becomes A 500**: `RecoveryMiddleware` turns a panic into
   the same generic 500 body, logged with a stack trace, and the server survives.
7. **The Content-Type Gate Guards Only Bodied Methods**: `GET` is never rejected;
   a `POST` declaring a non-JSON type is answered 415 before the handler runs;
   and a correct type with parameters attached is accepted, not refused on an
   exact-match technicality.
