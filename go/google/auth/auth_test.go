package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
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

	// 8. No authentication mode falls back to ADC.
	t.Run("No Authentication Mode falls back to ADC", func(t *testing.T) {
		t.Setenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT", "")
		opts, err := ResolveClient(ctx, nil)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(opts) != 0 {
			t.Errorf("expected empty ClientOptions for ADC fallback, got: %d", len(opts))
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
		err := v.ValidateAccess(ctx, "user@example.com")
		if err == nil {
			t.Fatal("expected error for nil validator, got nil")
		}
		if !strings.Contains(err.Error(), "validator or service is nil") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nil service in validator", func(t *testing.T) {
		v := NewDWDValidator(nil)
		err := v.ValidateAccess(ctx, "user@example.com")
		if err == nil {
			t.Fatal("expected error for nil service, got nil")
		}
		if !strings.Contains(err.Error(), "validator or service is nil") {
			t.Errorf("unexpected error: %v", err)
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

