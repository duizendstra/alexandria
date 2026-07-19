// Package auth provides core authentication builders, validations, and client resolution
// mechanisms for Google Workspace APIs.
//
// In Domain-Driven Design (DDD) terminology, the auth package acts as an Infrastructure
// Identity Translator. It decouples core domain services from low-level Google credential
// lifecycles, token storage, and authentication mechanics, producing standard
// option.ClientOption slices that can be injected into any Google API service constructor.
//
// # Bounded Context & Responsibilities
//
// 1. Identity Impersonation & Resolution: Resolves service account credentials, direct
// impersonation (SA-to-SA), and Domain-Wide Delegation (DWD).
//
// 2. Interactive Desktop Consent: Orchestrates local callback servers on random ports,
// authenticates via the default browser, loads/saves cached JSON tokens, and fetches client
// secret files securely from the macOS pass(1) utility.
//
// 3. Delegation Access Validation: Asserts domain delegations via DWDValidator, ensuring
// that credentials can successfully impersonate the target user with least-privilege metadata scopes.
//
// # SRE & Security Guardrails
//
// 1. Least-Privilege Scoping: Always request the narrowest required scope. For scans or
// inventory operations that only require metadata and not file content, prefer
// "https://www.googleapis.com/auth/drive.metadata.readonly" over the wider "drive.readonly".
//
// 2. Domain-Wide Delegation (DWD) Authorization: To authorize a Service Account for DWD in
// the Google Admin Console, use the following URL:
//
//	https://admin.google.com/ac/owl/domainwidedelegation?clientScopeToAdd=<scopes>&clientIdToAdd=<sa_client_id>&overwriteClientId=true
//
// When creating a client for DWD, always set the "Subject" field to the user email being impersonated.
//
// 3. Root Resolution Safety: Never rely on drive.About.RootFolderId to locate a user's root
// folder, as it is frequently empty or omitted in modern Google APIs. Always resolve the
// true root ID explicitly via:
//
//	service.Files.Get("root").Fields("id").Do()
//
// This pattern is built directly into DWDValidator.ValidateAccess.
//
// 4. API Resilience: Google API calls can be subject to rate-limiting (HTTP 429) or transient
// 5xx server errors. Every client resolved by ResolveClient routes HTTP traffic through a
// transport-level retry (github.com/duizendstra/alexandria/go/retry) with exponential backoff,
// so all API services built from the resolved options inherit a uniform retry policy. Requests
// with non-rewindable bodies (streaming uploads without GetBody) are sent exactly once. Tune the
// attempt budget with WithRetryAttempts or opt out with WithoutRetry. The DWDValidator
// additionally wraps its access check in GCP-aware retries
// (github.com/duizendstra/alexandria/go/retry/gcp) so it also protects services that were not
// built through ResolveClient.
//
// 5. Environment Fallbacks: If no explicit authentication configuration is supplied to
// ResolveClient, it dynamically falls back to the service account specified by the
// GOOGLE_IMPERSONATE_SERVICE_ACCOUNT environment variable, or finally to Google Application
// Default Credentials (ADC).
//
// # Storage & Caching
//
// Interactive consent tokens are securely cached locally under ~/.kratos/tokens/google-oauth.json.
// The client secrets are resolved via the Unix password store (`pass`) under the key
// "dui/google-oauth-client", or can be overridden via the GOOGLE_OAUTH_CLIENT environment variable.
package auth
