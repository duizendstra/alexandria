---
uuid: 9c5d1264-1377-46e4-a5c4-5c751d42b078
title: "Identity-Aware Proxy (IAP) & Split-Microservice Security Topology"
domain: "architecture"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-07-12T14:30:00Z"
updated_at: "2026-08-29T00:00:00Z"
summary: >
  Outlines the architectural topology, split-microservice boundary layout, and zero-trust security model
  for the GCP Cloud Run Native IAP and Google Groups authorization validation stack.
audience: [public]
tags: [ "gcp", "iap", "security", "microservices", "zero-trust" ]
relations:
  - target_uuid: "ee14bc6c-1349-411a-8bb4-f58c70a84e20" # DDD Boundaries
    rel_type: "extends"
---
# Identity-Aware Proxy (IAP) & Split-Microservice Security Topology

## Architectural Objective

To design a highly secure, scalable, and cost-effective multi-tenant application topology on Google Cloud Platform (GCP) that enforces strict zero-trust access control at the network edge, maintains clear privilege boundaries between standard user operations and high-privilege directory administration, and avoids expensive directory license requirements.

This architecture leverages GCP Native Identity-Aware Proxy (IAP) on Cloud Run to construct an impenetrable outer boundary, coupled with an innovative out-of-band parallel directory query pattern that isolates high-privilege Domain-Wide Delegation (DWD) access to a dedicated backend microservice.

---

## Split-Microservice Topology

The validation system is divided into two distinct logical services, each operating within its own security sandbox and running with separate GCP Service Accounts adhering to the principle of least privilege.

```
  [ External Client Request ]
               |
               v
  +--------------------------------------------------------+
  |             Google Cloud Edge (IAP Gate)               |
  |     - Performs mandatory OIDC / SAML Identity Check     |
  |     - Rejects unauthorized users at network edge       |
  |     - Injects X-Goog-Iap-Jwt-Assertion Header          |
  +---------------------------+----------------------------+
                              |
                     Verified | Request
                              v
  +--------------------------------------------------------+
  |              Service A: Core Frontend                  |
  |     - Standard User Dashboard / Read-Only Interface    |
  |     - Low-Privilege Service Account (No DWD)           |
  |     - Direct Group-Query Pattern (Parallel Resolution) |
  +---------------------------+----------------------------+
                              |
                              | High-Privilege Write Actions
                              v
  +--------------------------------------------------------+
  |               Service B: Admin API                     |
  |     - RESTful Unit / Membership Mutators               |
  |     - High-Privilege Service Account (DWD Enabled)     |
  |     - Impersonates Directory Admin for CRUD Operations |
  +--------------------------------------------------------+
```

### 1. Service A: Core Frontend (Low-Privilege Dashboard)
*   **Role**: Direct user entry point and dashboard interface.
*   **Ingress Protection**: Native Cloud Run IAP enabled. Unauthenticated traffic is blocked at the Google Edge before consumption of container resources.
*   **Identity Extraction**: Receives verified, cryptographically signed JWT assertions containing the user's authenticated identity and unique subject identifier.
*   **Authorization Scope**: Standard, read-only dashboard access. It is completely isolated from the Google Workspace administrative realm.

### 2. Service B: Admin API (High-Privilege Directory Mutator)
*   **Role**: Handles high-privilege REST API endpoints for unit creation, unit deletion, and staff/employee role allocations in the directory.
*   **Ingress Protection**: Private internal endpoint or restricted API gateway accessible only via Service-to-Service identity tokens.
*   **Authorization Scope**: Read-write access to the Google Workspace Directory API. This service possesses Domain-Wide Delegation (DWD) privileges, allowing it to impersonate directory administrators.

---

## Security Model & Access Control

### 1. Zero-Trust Edge Security (Native Cloud Run IAP)
Rather than executing authentication logic inside the container code, authentication is offloaded to Google's Global Load Balancing infrastructure.
*   **Edge Gatekeeping**: Cloud Run Native IAP intercepts incoming HTTP requests. If a request lacks a valid Google Session or an active OIDC token, the edge instantly triggers the identity flow.
*   **Group-Level Access Bounds**: Google Groups are bound directly to the Cloud Run IAM policy using the role `roles/iap.httpsResourceAccessor`. If a user is not a member of the designated Google Group, the edge rejects the connection with an HTTP 403 Forbidden page, protecting backend application compute from unauthorized traffic.

### 2. Direct Group-Query Pattern (DWD Bypass for Read Operations)
Standard approaches to listing a user's Google Groups memberships require Domain-Wide Delegation (DWD) to impersonate an administrator, or a paid Cloud Identity Premium license to query transitive member relations. This architecture implements a highly secure, non-DWD read pattern:
*   **Domain Listing**: The low-privilege Service Account uses its own identity to list the public group directory within the organization.
*   **Parallel Verification**: Instead of asking "which groups does user X belong to?" (which requires high privilege), the service iterates through the listed groups and queries "is user X a member of group Y?" in parallel.
*   **Purity of Boundary**: Because checking membership in a specific group is a lower-privilege API call, the Frontend service can verify a user's local group associations on-the-fly without needing any DWD capabilities, ensuring that compromised frontend sessions cannot execute administrative directory modifications.

### 3. Domain-Wide Delegation (DWD) Isolation
To prevent the high-privilege DWD authorization from leaking, all directory writes are completely segregated behind the Admin API microservice boundary.
*   **Impersonation Boundary**: The Admin API Service Account is the only principal authorized in the Google Workspace Admin Console to perform DWD.
*   **Subject Restrictions**: When performing actions via the Directory SDK, the Admin API explicitly configures the impersonation subject to target a restricted, non-super-admin administrative user, limiting the potential blast radius of a compromised API key or service account.
*   **Least-Privilege Scopes**: The service account scopes are locked down to the absolute minimum necessary (e.g. `admin.directory.group` and `admin.directory.group.member` instead of full directory-wide access).

---

## Architectural Rationale & Trade-offs

### 1. Edge-Level Ingress vs. In-App Authentication
*   **Pro**: Offloading identity checks to IAP ensures that application code does not deal with cryptographic handshake details, token refreshes, or user redirect logic, drastically reducing the potential surface area for authorization bypass bugs.
*   **Con**: Requires an enterprise GCP Organization structure, meaning standard free-tier Gmail accounts cannot be easily validated without configuring external identity provider setups.

### 2. Direct Group-Query Parallelism vs. Cloud Identity Premium
*   **Pro**: Bypasses the need for expensive per-user Cloud Identity licensing tiers while maintaining full security compliance.
*   **Con**: For organizations with thousands of distinct groups, listing and querying memberships in parallel could encounter Google API quota limitations (HTTP 429). The system mitigates this by applying a shared rate-limiter and utilizing exponential backoff on retry channels.
