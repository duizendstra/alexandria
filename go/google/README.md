# go/google

`go/google` provides secure, platform-aware Google Workspace authenticator builders and client factory constructors optimized for high-throughput cloud-native Go microservices.

## Features

- **Domain-Wide Delegation (DWD)**: Seamless OAuth2 impersonation of workspace users via target service accounts.
- **Fail-Fast DWD Validation**: Built-in `DWDValidator` checking access rights immediately on startup.
- **Service Account Impersonation**: Direct, secure SA-to-SA credentials configuration.
- **Interactive Consent Support**: Desktop-oriented consent flow with customizable token caching policies.
- **HTTP Transport Customization**: Injects custom HTTP clients to control connection limits, timeouts, and mocks.
- **Uniform Transient-Failure Retry**: Every client resolved by `auth.ResolveClient` routes HTTP traffic through a retrying transport (429/5xx with exponential backoff), so pagination crawls, downloads, and buffered uploads survive transient errors. Tune with `auth.WithRetryAttempts`, opt out with `auth.WithoutRetry`.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/google
```

## Quick Start

### 1. Initializing an Authenticated Google Drive Service

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/client"
)

func main() {
	ctx := context.Background()

	// Build a fully-authenticated Google Drive API client using DWD
	driveService, err := client.NewDriveService(ctx,
		auth.WithDomainWideDelegation("service-account@my-project.gserviceaccount.com", "user@my-domain.com"),
		auth.WithScopes("https://www.googleapis.com/auth/drive.readonly"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Google Drive client: %v", err)
	}

	fmt.Printf("Successfully created Drive service: %s\n", driveService.BasePath)
}
```

### 2. Validating Domain-Wide Delegation on Startup

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/client"
)

func main() {
	ctx := context.Background()

	driveService, err := client.NewDriveService(ctx,
		auth.WithDomainWideDelegation("service-account@my-project.gserviceaccount.com", "user@my-domain.com"),
		auth.WithScopes("https://www.googleapis.com/auth/drive.readonly"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	// Create validator to verify access before handling requests.
	// It validates the delegated subject the service credentials were
	// built with (user@my-domain.com above).
	validator := auth.NewDWDValidator(driveService)
	if err := validator.ValidateAccess(ctx); err != nil {
		log.Fatalf("DWD access check failed: %v", err)
	}

	fmt.Println("DWD access verified successfully")
}
```

## SRE & Performance Hardening details

1. **Fail-Fast Verification**: The `DWDValidator` validates authentication scopes and user delegation policies at startup, preventing cascading logical failures inside application hotpaths.
2. **Connection Pooling Preservation**: Custom HTTP client injection via `WithHTTPClient` lets services inject clients configured with specialized idle connection limits, preventing connection starvation.
3. **Impersonation Token Reuse**: Resolves access token generation using Google's native IAM credentials flow, avoiding hardcoded long-lived private keys.
