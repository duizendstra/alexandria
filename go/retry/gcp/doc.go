// Package gcp provides GCP/Google API specific error-classification and retry utilities.
//
// # What
//
// An error classification evaluator and execution wrapper that understands both
// REST-based googleapi.Error responses and standard gRPC status codes.
//
// # Who
//
// Used by GCP clients, Google Workspace integrations (like Google Drive or Admin SDK),
// and pipeline migrations to evaluate failures and perform backoffs.
//
// # When
//
// Wrap long-running API operations (like fetching directories, streaming logs, or scanning
// file trees) to protect against transient GCP server glitches (5xx) or rate limits (429).
//
// # Why
//
// GCP API failures can be caused by transient issues, rate limits, or permanent access denials.
// Centralizing this classification ensures client logic fails fast on authentication errors
// but gracefully handles network dropouts.
//
// Domain:  Retry
// Concern: How do we retry Google APIs safely?
package gcp
