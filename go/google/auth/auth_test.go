package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const notAnEmail = "not-an-email"

func TestMain(m *testing.M) {
	// Stub execCommand to prevent real subprocess execution/browser opening during tests.
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}
	os.Exit(m.Run())
}

func TestResolveClient_Validation(t *testing.T) {
	ctx := context.Background()

	// 1. ErrNoImpersonationAccount when impersonateSA is empty and env var is empty.
	t.Run("Empty ImpersonateSA and Empty Env", func(t *testing.T) {
		t.Setenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT", "")

		_, err := ResolveClient(ctx, nil, WithDomainWideDelegation("", "user@example.com"))
		if !errors.Is(err, ErrNoImpersonationAccount) {
			t.Errorf("expected ErrNoImpersonationAccount, got: %v", err)
		}
	})

	// 2. Env fallback is used and validated.
	t.Run("Env Fallback and Validate Succeeded/Failed", func(t *testing.T) {
		// Valid fallback but invalid format.
		t.Setenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT", "invalid-sa")

		_, err := ResolveClient(ctx, nil, WithDomainWideDelegation("", "user@example.com"))
		if !errors.Is(err, ErrInvalidServiceAccount) {
			t.Errorf("expected ErrInvalidServiceAccount, got: %v", err)
		}
	})

	// 3. ErrInvalidServiceAccount when impersonateSA has invalid formats.
	t.Run("Invalid Service Account Formats", func(t *testing.T) {
		invalidSAs := []string{
			notAnEmail,
			"missing-at.gserviceaccount.com",
			"too@many@ats.gserviceaccount.com",
			"spaces in@domain.gserviceaccount.com",
			"@project-id.iam.gserviceaccount.com",
		}

		for _, sa := range invalidSAs {
			t.Run(sa, func(t *testing.T) {
				_, err := ResolveClient(ctx, nil, WithDomainWideDelegation(sa, "user@example.com"))
				if !errors.Is(err, ErrInvalidServiceAccount) {
					t.Errorf("for service account %q: expected ErrInvalidServiceAccount, got: %v", sa, err)
				}
			})
		}
	})
}

func TestResolveClient_SubjectValidation(t *testing.T) {
	ctx := context.Background()

	// 4. ErrNoSubjectEmail when subjectEmail is empty.
	t.Run("Empty Subject Email", func(t *testing.T) {
		_, err := ResolveClient(ctx, nil, WithDomainWideDelegation("sa@project.iam.gserviceaccount.com", ""))
		if !errors.Is(err, ErrNoSubjectEmail) {
			t.Errorf("expected ErrNoSubjectEmail, got: %v", err)
		}
	})

	// 5. ErrInvalidSubjectEmail when subjectEmail has invalid formats.
	t.Run("Invalid Subject Email Formats", func(t *testing.T) {
		invalidSubjects := []string{
			notAnEmail,
			"missing-at.com",
			"too@many@ats.com",
			"spaces in@domain.com",
			"@domain.com",
			"user@",
		}

		for _, sub := range invalidSubjects {
			t.Run(sub, func(t *testing.T) {
				_, err := ResolveClient(ctx, nil, WithDomainWideDelegation("sa@project.iam.gserviceaccount.com", sub))
				if !errors.Is(err, ErrInvalidSubjectEmail) {
					t.Errorf("for subject %q: expected ErrInvalidSubjectEmail, got: %v", sub, err)
				}
			})
		}
	})
}

func TestResolveClient_Resolution(t *testing.T) {
	ctx := context.Background()

	// 6. Valid parameters bypass fast-fail validation and attempt authentication.
	t.Run("Valid Inputs Pass Fast-Fail", func(t *testing.T) {
		// Valid service account and subject email.
		_, err := ResolveClient(ctx, nil, WithDomainWideDelegation("sa@project.iam.gserviceaccount.com", "user@domain.com"))

		// It should bypass validation and fail inside credentials/token source lookup.
		if errors.Is(err, ErrNoImpersonationAccount) ||
			errors.Is(err, ErrNoSubjectEmail) ||
			errors.Is(err, ErrInvalidServiceAccount) ||
			errors.Is(err, ErrInvalidSubjectEmail) {
			t.Errorf("expected validation to pass, but got validation error: %v", err)
		}

		if err == nil {
			t.Log("Successfully initialized or bypassed validation")
		} else {
			t.Logf("Passed fast-fail validation. Downstream error as expected: %v", err)
			if !strings.Contains(err.Error(), "failed to create impersonated credentials") {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	// 7. Inject custom client.
	t.Run("Inject Custom HTTP Client", func(t *testing.T) {
		client := &http.Client{}
		opts, err := ResolveClient(ctx, nil, WithHTTPClient(client))
		if err != nil {
			t.Fatalf("expected no error with custom HTTP client injection, got: %v", err)
		}
		if len(opts) == 0 {
			t.Fatal("expected non-empty ClientOption slice")
		}
	})

	// 8. No authentication mode falls back to ADC. With retry disabled the
	// resolution is deferred to the API service constructor (empty options).
	t.Run("No Authentication Mode falls back to deferred ADC without retry", func(t *testing.T) {
		t.Setenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT", "")
		opts, err := ResolveClient(ctx, nil, WithoutRetry())
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(opts) != 0 {
			t.Errorf("expected empty ClientOptions for ADC fallback, got: %d", len(opts))
		}
	})

	// 9. With retry enabled (the default), the ADC fallback resolves eagerly
	// into a single retrying HTTP client option.
	t.Run("No Authentication Mode builds retrying ADC client", func(t *testing.T) {
		t.Setenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT", "")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", writeFakeServiceAccountKey(t))

		opts, err := ResolveClient(ctx, []string{"https://www.googleapis.com/auth/drive.metadata.readonly"})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(opts) != 1 {
			t.Errorf("expected a single retrying HTTP client option for ADC fallback, got: %d", len(opts))
		}
	})
}

func TestValidationFunctions(t *testing.T) {
	t.Run("IsValidEmail", func(t *testing.T) {
		valid := []string{"test@example.com", "sa@project.iam.gserviceaccount.com", "test@example"}
		for _, email := range valid {
			if !IsValidEmail(email) {
				t.Errorf("expected %q to be a valid email", email)
			}
		}

		invalid := []string{notAnEmail, "test@", "@example.com", "test @example.com"}
		for _, email := range invalid {
			if IsValidEmail(email) {
				t.Errorf("expected %q to be an invalid email", email)
			}
		}
	})

	t.Run("IsValidServiceAccount", func(t *testing.T) {
		if !IsValidServiceAccount("sa@project.gserviceaccount.com") {
			t.Error("expected sa@project.gserviceaccount.com to be valid service account")
		}
		if !IsValidServiceAccount("user@example.com") {
			t.Error("expected user@example.com to be valid service account format")
		}
	})
}

func TestNewDWDValidator(t *testing.T) {
	t.Run("creates validator with non-nil service", func(t *testing.T) {
		srv := &drive.Service{}
		val := NewDWDValidator(srv)
		if val == nil {
			t.Fatal("expected non-nil DWDValidator")
		}
		if val.service != srv {
			t.Error("expected validator service to match input service")
		}
	})
}

func TestResolveClient_InteractiveConsent(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid passKey with hyphen", func(t *testing.T) {
		t.Setenv("GOOGLE_OAUTH_CLIENT", "") // Ensure we don't bypass with env var.
		_, err := ResolveClient(ctx, nil, WithInteractiveConsent("-invalid-key", ""))
		if err == nil {
			t.Fatal("expected error for hyphenated passKey, got nil")
		}
		if !strings.Contains(err.Error(), "invalid pass key: cannot start with a hyphen") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("default interactive option activation", func(t *testing.T) {
		// Calling WithInteractiveConsent("", "") should activate the interactive flow
		// and attempt to resolve credentials using default passKey, not fall back to ADC.
		t.Setenv("GOOGLE_OAUTH_CLIENT", "")
		_, err := ResolveClient(ctx, nil, WithInteractiveConsent("", ""))
		if err == nil {
			t.Fatal("expected error due to missing default passKey command, got nil")
		}
		// Since "pass" command is likely not present in the test environment or key is missing,
		// we expect it to fail in resolveCredentials rather than returning nil (ADC).
		if strings.Contains(err.Error(), "no Google authentication mode was configured") {
			t.Errorf("expected interactive flow to be invoked, but it fell back to ADC")
		}
	})
}

func TestDWDValidator_ValidateAccess_NilService(t *testing.T) {
	ctx := context.Background()

	t.Run("nil validator", func(t *testing.T) {
		var v *DWDValidator
		err := v.ValidateAccess(ctx)
		if err == nil {
			t.Fatal("expected error for nil validator, got nil")
		}
		if !strings.Contains(err.Error(), "validator or service is nil") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nil service in validator", func(t *testing.T) {
		v := NewDWDValidator(nil)
		err := v.ValidateAccess(ctx)
		if err == nil {
			t.Fatal("expected error for nil service, got nil")
		}
		if !strings.Contains(err.Error(), "validator or service is nil") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// writeFakeServiceAccountKey writes a syntactically valid service account key
// file (with a freshly generated RSA key) and returns its path. Credential
// detection parses the file without any network calls.
func writeFakeServiceAccountKey(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	sa := map[string]string{
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "fake-key-id",
		"private_key":    string(pemBytes),
		"client_email":   "fake@test-project.iam.gserviceaccount.com",
		"client_id":      "1234567890",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}

	data, err := json.Marshal(sa)
	if err != nil {
		t.Fatalf("marshal service account JSON: %v", err)
	}

	path := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write service account file: %v", err)
	}

	return path
}

// flakyDriveServer returns an httptest server that answers Drive-style
// files.get requests, failing with failStatus for the first failures requests
// and succeeding afterwards. The returned counter tracks total requests.
func flakyDriveServer(t *testing.T, failures, failStatus int) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	calls := new(atomic.Int64)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= int64(failures) {
			http.Error(w, `{"error":{"code":429,"message":"rate limited"}}`, failStatus)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"root"}`))
	}))
	t.Cleanup(ts.Close)

	return ts, calls
}

// resolveDriveService builds a Drive service against the test server using the
// options produced by ResolveClient.
func resolveDriveService(t *testing.T, ctx context.Context, endpoint string, opts ...Option) *drive.Service {
	t.Helper()

	clientOpts, err := ResolveClient(ctx, nil, opts...)
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}

	srv, err := drive.NewService(ctx, append(clientOpts, option.WithEndpoint(endpoint))...)
	if err != nil {
		t.Fatalf("drive.NewService: %v", err)
	}

	return srv
}

func TestResolveClient_TransportRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("retries transient failures and succeeds", func(t *testing.T) {
		ts, calls := flakyDriveServer(t, 2, http.StatusTooManyRequests)
		srv := resolveDriveService(t, ctx, ts.URL, WithHTTPClient(&http.Client{}))

		file, err := srv.Files.Get("root").Fields("id").Context(ctx).Do()
		if err != nil {
			t.Fatalf("expected retried call to succeed, got: %v", err)
		}
		if file.Id != "root" {
			t.Errorf("expected file ID root, got %q", file.Id)
		}
		if calls.Load() != 3 {
			t.Errorf("expected 3 attempts (2 failures + 1 success), got %d", calls.Load())
		}
	})

	t.Run("WithoutRetry performs a single attempt", func(t *testing.T) {
		ts, calls := flakyDriveServer(t, 1, http.StatusInternalServerError)
		srv := resolveDriveService(t, ctx, ts.URL, WithHTTPClient(&http.Client{}), WithoutRetry())

		if _, err := srv.Files.Get("root").Fields("id").Context(ctx).Do(); err == nil {
			t.Fatal("expected error without retry, got nil")
		}
		if calls.Load() != 1 {
			t.Errorf("expected exactly 1 attempt, got %d", calls.Load())
		}
	})

	t.Run("WithRetryAttempts caps total attempts", func(t *testing.T) {
		ts, calls := flakyDriveServer(t, 1000, http.StatusServiceUnavailable)
		srv := resolveDriveService(t, ctx, ts.URL, WithHTTPClient(&http.Client{}), WithRetryAttempts(2))

		if _, err := srv.Files.Get("root").Fields("id").Context(ctx).Do(); err == nil {
			t.Fatal("expected error after exhausting retries, got nil")
		}
		if calls.Load() != 2 {
			t.Errorf("expected exactly 2 attempts, got %d", calls.Load())
		}
	})

	t.Run("does not mutate the injected client", func(t *testing.T) {
		client := &http.Client{}
		if _, err := ResolveClient(ctx, nil, WithHTTPClient(client)); err != nil {
			t.Fatalf("ResolveClient: %v", err)
		}
		if client.Transport != nil {
			t.Error("expected the caller-owned client to remain untouched")
		}
	})
}

func TestOpenBrowserSafety(t *testing.T) {
	ctx := context.Background()

	t.Run("unsafe HTTP scheme", func(t *testing.T) {
		err := openBrowser(ctx, "http://example.com/oauth")
		if err == nil {
			t.Fatal("expected error for unsafe http scheme, got nil")
		}
		if !strings.Contains(err.Error(), "refusing to open URL: unsafe scheme or protocol") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-url command injection attempt", func(t *testing.T) {
		err := openBrowser(ctx, "file:///etc/passwd")
		if err == nil {
			t.Fatal("expected error for non-https protocol, got nil")
		}
	})
}

