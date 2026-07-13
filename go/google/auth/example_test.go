package auth_test

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/duizendstra/alexandria/go/google/auth"
	"google.golang.org/api/drive/v3"
)

// ExampleResolveClient_serviceAccountImpersonation demonstrates how to resolve Google client options
// using standard Service Account-to-Service Account impersonation. This is ideal for secure,
// keyless machine-to-machine integrations.
func ExampleResolveClient_serviceAccountImpersonation() {
	ctx := context.Background()
	targetSA := "my-app-worker@my-gcp-project.iam.gserviceaccount.com"

	// Resolve standard option.ClientOption list
	clientOpts, err := auth.ResolveClient(
		ctx,
		[]string{drive.DriveMetadataReadonlyScope},
		auth.WithServiceAccountImpersonation(targetSA),
	)
	if err != nil {
		fmt.Printf("failed to resolve client options: %v\n", err)
		return
	}

	// The clientOpts can now be passed directly into any standard Google API service constructor
	srv, err := drive.NewService(ctx, clientOpts...)
	if err != nil {
		fmt.Printf("failed to create drive service: %v\n", err)
		return
	}

	_ = srv
}

// ExampleResolveClient_domainWideDelegation demonstrates how to resolve Google client options using
// Domain-Wide Delegation (DWD). DWD allows a service account to impersonate a specific user
// within a Google Workspace domain.
func ExampleResolveClient_domainWideDelegation() {
	ctx := context.Background()
	targetSA := "workspace-scanner@my-gcp-project.iam.gserviceaccount.com"
	impersonatedUser := "ceo@my-company.com"

	// Resolve client options with DWD configuration and least-privilege metadata scope
	clientOpts, err := auth.ResolveClient(
		ctx,
		[]string{drive.DriveMetadataReadonlyScope},
		auth.WithDomainWideDelegation(targetSA, impersonatedUser),
	)
	if err != nil {
		fmt.Printf("failed to resolve DWD client options: %v\n", err)
		return
	}

	srv, err := drive.NewService(ctx, clientOpts...)
	if err != nil {
		fmt.Printf("failed to create drive service: %v\n", err)
		return
	}

	_ = srv
}

// ExampleResolveClient_interactiveConsent demonstrates how to resolve Google client options using
// interactive, desktop-based OAuth2 flows. This flow relies on a local Unix pass store for secrets
// and caches tokens locally.
//
// Note: This example does not include an Output assertion comment, keeping it as a "compile-only"
// test. This ensures CI can verify its correctness without running the actual browser flow.
func ExampleResolveClient_interactiveConsent() {
	ctx := context.Background()
	passKey := "my-org/google-oauth-client"
	tokenPath := ".tokens/my-cached-token.json"

	// Resolve client options using local browser-based authentication flow.
	// We inject a default logger to observe the authorization redirect messages.
	clientOpts, err := auth.ResolveClient(
		ctx,
		[]string{drive.DriveMetadataReadonlyScope},
		auth.WithInteractiveConsent(passKey, tokenPath),
		auth.WithLogger(slog.Default()),
	)
	if err != nil {
		// In a headless CI environment, this may fail if "pass" is missing or non-interactive.
		// We handle it gracefully in the example.
		fmt.Printf("interactive flow initialized or failed gracefully: %v\n", err)
		return
	}

	_ = clientOpts
	fmt.Println("Interactive client resolved successfully")
}

// ExampleDWDValidator_ValidateAccess demonstrates how to use the DWDValidator to verify that
// Domain-Wide Delegation (DWD) is active and authorized for a target subject email.
//
// Note: This example does not include an Output assertion comment to keep it as a "compile-only"
// test, preventing network-bound failures in offline or headless environments.
func ExampleDWDValidator_ValidateAccess() {
	ctx := context.Background()

	// 1. Resolve credentials for DWD
	clientOpts, err := auth.ResolveClient(
		ctx,
		[]string{drive.DriveMetadataReadonlyScope},
		auth.WithDomainWideDelegation(
			"workspace-scanner@my-gcp-project.iam.gserviceaccount.com",
			"ceo@my-company.com",
		),
	)
	if err != nil {
		fmt.Printf("failed to resolve client: %v\n", err)
		return
	}

	// 2. Instantiate Google Drive Service under impersonated user context
	srv, err := drive.NewService(ctx, clientOpts...)
	if err != nil {
		fmt.Printf("failed to create drive service: %v\n", err)
		return
	}

	// 3. Create the validator with the drive service
	validator := auth.NewDWDValidator(srv)

	// 4. Assert that we can access root metadata (e.g., verifying delegation is active).
	// This performs a Files.Get("root") call wrapped in exponential backoff.
	err = validator.ValidateAccess(ctx, "ceo@my-company.com")
	if err != nil {
		fmt.Printf("DWD validation failed: %v\n", err)
		return
	}

	fmt.Println("DWD authorization validated successfully")
}
