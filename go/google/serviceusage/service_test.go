package serviceusage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/google/serviceusage"
	"google.golang.org/api/option"
	serviceusageapi "google.golang.org/api/serviceusage/v1"
)

const (
	testProjectID = "my-test-proj"
	driveAPI      = "drive.googleapis.com"
	iamAPI        = "iam.googleapis.com"
	crmAPI        = "cloudresourcemanager.googleapis.com"
)

type mockUsageHandler struct {
	opPollCount atomic.Int64
	failOp      atomic.Bool
}

func (h *mockUsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// List enabled services.
	case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/"+testProjectID+"/services":
		resp := &serviceusageapi.ListServicesResponse{
			Services: []*serviceusageapi.GoogleApiServiceusageV1Service{
				{
					Config: &serviceusageapi.GoogleApiServiceusageV1ServiceConfig{
						Name: driveAPI,
					},
					State: "ENABLED",
				},
				{
					Config: &serviceusageapi.GoogleApiServiceusageV1ServiceConfig{
						Name: iamAPI,
					},
					State: "ENABLED",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson // mock response

	// Batch enable services.
	case r.Method == http.MethodPost && r.URL.Path == "/v1/projects/"+testProjectID+"/services:batchEnable":
		op := &serviceusageapi.Operation{
			Name: "operations/batch-enable-123",
			Done: false,
		}
		_ = json.NewEncoder(w).Encode(op) //nolint:errchkjson // mock response

	// Operation status poll.
	case r.Method == http.MethodGet && r.URL.Path == "/v1/operations/batch-enable-123":
		count := h.opPollCount.Add(1)
		done := count >= 2
		op := &serviceusageapi.Operation{
			Name: "operations/batch-enable-123",
			Done: done,
		}
		if done && h.failOp.Load() {
			op.Error = &serviceusageapi.Status{
				Code:    7,
				Message: "permission denied for api enablement",
			}
		}
		_ = json.NewEncoder(w).Encode(op) //nolint:errchkjson // mock response

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestService_APIEnablement(t *testing.T) {
	t.Parallel()

	handler := &mockUsageHandler{}
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	ctx := context.Background()
	cfg := serviceusage.Config{
		PollInterval: 10 * time.Millisecond,
		MaxAttempts:  5,
	}

	svc, err := serviceusage.New(
		ctx,
		cfg,
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("serviceusage.New failed: %v", err)
	}

	// 1. Check API status.
	status, err := svc.CheckAPIsStatus(ctx, testProjectID, []string{
		driveAPI,
		iamAPI,
		crmAPI,
	})
	if err != nil {
		t.Fatalf("CheckAPIsStatus failed: %v", err)
	}

	if !status[driveAPI] {
		t.Errorf("expected %s to be enabled", driveAPI)
	}
	if !status[iamAPI] {
		t.Errorf("expected %s to be enabled", iamAPI)
	}
	if status[crmAPI] {
		t.Errorf("expected %s to NOT be enabled", crmAPI)
	}

	// 2. Validation.
	_, err = svc.CheckAPIsStatus(ctx, "", []string{driveAPI})
	if !errors.Is(err, serviceusage.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID, got: %v", err)
	}
	err = svc.BatchEnableAPIs(ctx, "", []string{driveAPI})
	if !errors.Is(err, serviceusage.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID for BatchEnableAPIs, got: %v", err)
	}

	// 3. Empty APIs list.
	if err := svc.BatchEnableAPIs(ctx, testProjectID, nil); err != nil {
		t.Errorf("empty APIs should succeed immediately, got: %v", err)
	}

	// 4. Batch Enable APIs with successful polling.
	err = svc.BatchEnableAPIs(ctx, testProjectID, []string{crmAPI})
	if err != nil {
		t.Fatalf("BatchEnableAPIs failed: %v", err)
	}

	// 5. Batch Enable APIs with operation error.
	handler.failOp.Store(true)
	handler.opPollCount.Store(0)
	err = svc.BatchEnableAPIs(ctx, testProjectID, []string{crmAPI})
	if err == nil {
		t.Fatal("expected operation error, got nil")
	}
	if !errors.Is(err, serviceusage.ErrAPIEnablementFailed) {
		t.Fatalf("expected ErrAPIEnablementFailed, got: %v", err)
	}
}
