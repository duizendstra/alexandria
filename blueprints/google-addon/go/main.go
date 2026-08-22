package main

import (
	"log/slog"
	"net/http"
	"os"
	_ "time/tzdata" // Required for static container base images

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
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

	slog.Info("starting Google Calendar Add-on service", "port", port)
	if err := http.ListenAndServe(":"+port, sloggcp.TraceMiddleware(mux)); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed to start", "err", err)
		os.Exit(1)
	}
}
