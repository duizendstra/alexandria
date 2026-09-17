package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "time/tzdata" // Required for static container base images.

	"github.com/duizendstra/alexandria/go/platform/web"
	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

// Cloud Run allows a request timeout of up to 60 minutes, so a service that sets
// none of its own has effectively unbounded reads and writes.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

func main() {
	sloggcp.Setup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", HandleCalendarTrigger)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           sloggcp.TraceMiddleware(web.RecoveryMiddleware(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	slog.Info("starting Google Calendar Add-on service", slog.String("port", port))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed to start", slog.Any("err", err))
		os.Exit(1)
	}
}
