---
uuid: b4bc306c-9ba5-4eb8-b99b-efb829623dc1
title: "ADR-0002: Vault-Centric Documentation Structure"
domain: "decisions"
type: "architecture_decision_record"
diataxis_quadrant: "explanation"
status: "accepted"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-03T09:00:00Z"
updated_at: "2026-07-12T14:30:00Z"
summary: >
  Establishes a unified, OKF-compliant documentation vault to organize
  architectural, governance, and operational knowledge.
audience: [public]
tags: [ "adr", "documentation", "structure" ]
relations: []
---
# ADR-0002: Vault-Centric Documentation Structure

## Status

Accepted

## Context

Engineering documentation across our various codebases was highly fragmented. Context, standards, and guides were scattered across unstructured root directories, ad-hoc wiki systems, and team-internal repositories. 

This fragmentation created severe systemic challenges:
1.  **High Search Friction** — Developers could not easily locate authoritative guides, leading to duplication of research effort.
2.  **Lack of Semantic Context** — Modern engineering tools and agentic AI systems (LLMs) could not parse, filter, or traverse the documentation context due to a lack of standard formatting and metadata schemas.
3.  **Rapid Information Decay** — Documentation was rarely kept in sync with actual code implementations, causing setups and runbooks to break quickly.

## Decision

We establish a unified, structured **Documentation Vault** directly within our shared codebase root under `/docs` following Google's **Open Knowledge Format (OKF)** specification and the **Diátaxis framework**:

1.  **Standard-Based Directory layout** — Organize all documents into eight dedicated numbered domains:
    *   `01-governance/` — Legal, charters, team rosters, and risk matrices.
    *   `02-strategy/` — Roadmap deliverables and capability matrices.
    *   `03-architecture/` — System flows, layer boundaries, and tactical DDD patterns.
    *   `04-decisions/` — Architecture Decision Records (ADRs) using the MADR format.
    *   `05-security/` — Cryptographic baselines, IAM policies, and access controls.
    *   `06-operations/` — Deployment recipes, runbooks, and recovery checklists.
    *   `07-playbooks/` — Step-by-step developer tutorials (onboarding).
    *   `08-reference/` — Ubiquitous Language glossaries, dictionary models, and API references.
2.  **Standard Metadata Schema** — Every document must declare a RFC-compliant YAML frontmatter header defining standard metadata fields: `uuid` (RFC 4122 v4), `title`, `domain` (bounded context), `type` (Diátaxis quadrant), `status`, `maturity`, `owner`, `created_at`, `updated_at`, `summary` (dense text digest to optimize semantic vector-retrieval), `audience`, `tags`, and `relations`.
3.  **Automated Schema Linters** — Integrate a custom documentation validator into our CLI toolchain to audit the docs directory during CI/CD to prevent drift, catch duplicate UUIDs, and block broken relations.

## Consequences

### Easier
*   **Drift Prevention** — Stale docs, dead links, or invalid headers will instantly fail the build, forcing documentation to remain a first-class citizen alongside code.
*   **Predictable Discovery** — Human developers and AI context-retrieval systems can traverse and read authoritative knowledge from a single, highly structured, and queryable graph.

### Harder
*   **Slight Creation Overhead** — Authors must generate an immutable UUIDv4 and declare all standard frontmatter fields for every new document, adding a step to the editing process.
