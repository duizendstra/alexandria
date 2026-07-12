---
uuid: 91bc6c63-db3d-4c31-90be-e0dfc3df2220
title: "Zero-Trust Workload Security & Credential Hygiene"
domain: "security"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-03T09:00:00Z"
updated_at: "2026-07-12T14:30:00Z"
summary: >
  Defines the standard data classification, secure secret management,
  and keyless cloud authentication policies enforced across all modules.
audience: [public]
tags: [ "security", "policy", "secrets", "workloads" ]
relations: []
---
# Zero-Trust Workload Security & Credential Hygiene

> **Classification: INTERNAL**
> This policy establishes the secure credential injection and keyless authorization standards across our software ecosystem.

## Core Security Rules

### 1. Zero Static Credentials
*   **Static Keys are Prohibited** — Standing, unencrypted JSON service account credentials, access keys, database passwords, or OAuth client secrets must **never** be checked into git, written in configurations, or stored on developers' workstations.
*   **Keyless Workload Authentication** — Runtime processes executing in cloud environments must authenticate utilizing OIDC and Workload Identity (federated assertions). Standalone service identities exchange short-lived tokens on-demand.

### 2. Standard Secret Injection (pass)
*   **Declarative Local Injection** — Local development credentials must be encrypted in a centralized GPG standard password store (such as `pass`). These credentials are dynamically loaded into the environment at shell startup, avoiding persistent configuration files.
*   **Cloud Secret Management** — Production services load configurations directly from a centralized key vault (such as Google Cloud Secret Manager) at execution time.

### 3. Log Payload Masking
*   **Multi-Channel Masking** — Standard structured log handlers (`slog-gcp`) must dynamically scan and redact sensitive key-value pairs (e.g., `Authorization`, `bearer`, `access_token`, `password`, `ssn`).

---

## Data Classification Matrix

| Category | Definition | Protection Requirements |
|---|---|---|
| **Public** | Open-source code, schemas, and public API interfaces. | Fully unencrypted Git. |
| **Internal** | Repository codebases, system diagrams, and team roadmaps. | Branch permission policies and mandatory peer reviews. |
| **Confidential** | Live configurations, access tokens, and synthetic test fixtures. | GPG-encrypted password stores (`pass`) or Vault. |
| **Restricted** | Customer PII, raw compliance reports, and audit logs. | KMS encryption at rest (CMEK) and strict access-logging. |

---

## Remediation Protocol

In the event that an API token, private key, or password is accidentally committed:
1.  **Revoke Instantly** — Invalidate and rotate the active credential immediately to eliminate exploit vectors.
2.  **Purge Repository History** — Execute `git filter-repo` to permanently remove the commit and content from the physical Git object database. Standard `git rm` is insufficient.
3.  **Document and Post-Mortem** — File an ADR under `docs/04-decisions/` capturing the vulnerability root cause and preventative adjustments.
