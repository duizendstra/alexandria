// Package drive integrates with the Google Drive API under the Google Workspace SaaS Bounded Context.
//
// Domain:  Integration / Google Workspace
// Concern: How do we list, find, download, and upload files on Google Drive?
// Status:  Production-Ready SRE-grade
//
// # Design Grounding (Architectural Decisions)
//
// 1. Bounded Context Separation: This package is placed under `go/google/workspace/drive` instead of
// root `go/google/` or `go/integration/` to explicitly isolate enterprise productivity SaaS endpoints
// (Google Workspace) from core Google Cloud Platform (GCP) infrastructure clients.
//
// 2. High-Performance Streaming Scanner: The Scanner uses a functional streaming callback pattern
// `func(*drive.File) error` instead of returning buffered arrays. This guarantees O(1) memory complexity,
// ensuring safety when scanning directories with millions of items during large-scale migrations.
//
// # SRE Guardrails & Operational Constraints
//
// 1. Pagination Management: Never fetch massive datasets in a single HTTP call. Sane pagination defaults
// are set to 100 items per page (maximum: 1000). Use WithPageSize to override.
//
// 2. Least-Privilege Scopes: When initializing the client, always use the narrowest possible OAuth scope.
// For metadata-only listing operations (such as scans), prefer `drive.DriveMetadataReadonlyScope`.
// Reserve `drive.DriveReadonlyScope` for downloading actual file content.
//
// 3. Domain-Wide Delegation (DWD): This package fully supports DWD, allowing SAs to impersonate specific
// target users. Provide the Workspace Super Administrator with the following direct link to authorize
// the client ID and scopes in the Google Workspace Admin console:
//
//	https://admin.google.com/ac/owl/domainwidedelegation?clientScopeToAdd=https://www.googleapis.com/auth/drive.metadata.readonly,https://www.googleapis.com/auth/drive.readonly&clientIdToAdd=[YOUR_CLIENT_ID]&overwriteClientId=true
//
// 4. Context Propagation: All network operations accept a context.Context. Ensure your orchestrators
// propagate context cancellation and deadlines to prevent thread exhaustion or command-line freezes.
package drive
