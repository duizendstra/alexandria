package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/client"
)

func TestNewServices_InjectClient(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{}

	t.Run("NewDriveService", func(t *testing.T) {
		srv, err := client.NewDriveService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil drive service")
		}
	})

	t.Run("NewAdminService", func(t *testing.T) {
		srv, err := client.NewAdminService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil admin service")
		}
	})

	t.Run("NewReportsService", func(t *testing.T) {
		srv, err := client.NewReportsService(ctx, auth.WithHTTPClient(httpClient))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected non-nil reports service")
		}
	})
}
