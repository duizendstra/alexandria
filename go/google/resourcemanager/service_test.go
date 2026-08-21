package resourcemanager_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/google/resourcemanager"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

const (
	testProjectID      = "my-test-proj"
	testBillingAccount = "012345-6789AB-CDEF01"
)

type mockCRMHandler struct {
	projectGetCalls atomic.Int64
	projectExists   atomic.Bool
	billingLinked   atomic.Bool
}

func (h *mockCRMHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// CRM: Get project.
	case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/"+testProjectID:
		h.projectGetCalls.Add(1)
		if !h.projectExists.Load() {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson // mock response
				"error": map[string]any{
					"code":    404,
					"message": "project not found",
				},
			})

			return
		}

		_ = json.NewEncoder(w).Encode(&cloudresourcemanager.Project{ //nolint:errchkjson // mock response
			ProjectId:      testProjectID,
			LifecycleState: "ACTIVE",
		})

	// CRM: Create project.
	case r.Method == http.MethodPost && r.URL.Path == "/v1/projects":
		h.projectExists.Store(true)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(&cloudresourcemanager.Project{ //nolint:errchkjson // mock response
			ProjectId:      testProjectID,
			LifecycleState: "ACTIVE",
		})

	// Billing: Get billing info.
	case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/"+testProjectID+"/billingInfo":
		linked := h.billingLinked.Load()
		accountName := ""
		if linked {
			accountName = "billingAccounts/" + testBillingAccount
		}
		_ = json.NewEncoder(w).Encode(&cloudbilling.ProjectBillingInfo{ //nolint:errchkjson // mock response
			ProjectId:          testProjectID,
			BillingEnabled:     linked,
			BillingAccountName: accountName,
		})

	// Billing: Update billing info.
	case r.Method == http.MethodPut && r.URL.Path == "/v1/projects/"+testProjectID+"/billingInfo":
		h.billingLinked.Store(true)
		_ = json.NewEncoder(w).Encode(&cloudbilling.ProjectBillingInfo{ //nolint:errchkjson // mock response
			ProjectId:          testProjectID,
			BillingEnabled:     true,
			BillingAccountName: "billingAccounts/" + testBillingAccount,
		})

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestService_ProjectLifecycle(t *testing.T) {
	t.Parallel()

	handler := &mockCRMHandler{}
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	ctx := context.Background()
	cfg := resourcemanager.Config{
		PollInterval: 10 * time.Millisecond,
		MaxAttempts:  5,
	}

	svc, err := resourcemanager.New(
		ctx,
		cfg,
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("resourcemanager.New failed: %v", err)
	}

	// 1. Check exists when it does not.
	exists, err := svc.CheckProjectExists(ctx, testProjectID)
	if err != nil {
		t.Fatalf("CheckProjectExists failed: %v", err)
	}
	if exists {
		t.Errorf("expected project not to exist initially")
	}

	// 2. Validation errors.
	_, err = svc.CheckProjectExists(ctx, "")
	if !errors.Is(err, resourcemanager.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID, got: %v", err)
	}
	err = svc.CreateProject(ctx, resourcemanager.ProjectRequest{ProjectID: ""})
	if !errors.Is(err, resourcemanager.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID for create, got: %v", err)
	}

	// 3. Create project (which sets projectExists = true and polls).
	err = svc.CreateProject(ctx, resourcemanager.ProjectRequest{
		ProjectID: testProjectID,
		Name:      "My Test Project",
		OrgID:     "123456789",
	})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// 4. Verify exists returns true now.
	exists, err = svc.CheckProjectExists(ctx, testProjectID)
	if err != nil {
		t.Fatalf("CheckProjectExists failed: %v", err)
	}
	if !exists {
		t.Errorf("expected project to exist after creation")
	}

	// 5. Billing check (not linked yet).
	linked, err := svc.CheckBillingLinked(ctx, testProjectID, testBillingAccount)
	if err != nil {
		t.Fatalf("CheckBillingLinked failed: %v", err)
	}
	if linked {
		t.Errorf("expected billing not to be linked yet")
	}

	// 6. Link billing.
	err = svc.LinkBilling(ctx, testProjectID, testBillingAccount)
	if err != nil {
		t.Fatalf("LinkBilling failed: %v", err)
	}

	// 7. Verify billing linked returns true.
	linked, err = svc.CheckBillingLinked(ctx, testProjectID, testBillingAccount)
	if err != nil {
		t.Fatalf("CheckBillingLinked after update failed: %v", err)
	}
	if !linked {
		t.Errorf("expected billing to be linked after update")
	}

	// 8. Billing validation.
	_, err = svc.CheckBillingLinked(ctx, "", testBillingAccount)
	if !errors.Is(err, resourcemanager.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID, got: %v", err)
	}
	_, err = svc.CheckBillingLinked(ctx, testProjectID, "")
	if !errors.Is(err, resourcemanager.ErrEmptyBillingAccountID) {
		t.Errorf("expected ErrEmptyBillingAccountID, got: %v", err)
	}
	err = svc.LinkBilling(ctx, "", testBillingAccount)
	if !errors.Is(err, resourcemanager.ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID for LinkBilling, got: %v", err)
	}
	err = svc.LinkBilling(ctx, testProjectID, "")
	if !errors.Is(err, resourcemanager.ErrEmptyBillingAccountID) {
		t.Errorf("expected ErrEmptyBillingAccountID for LinkBilling, got: %v", err)
	}
}
