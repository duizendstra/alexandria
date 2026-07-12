---
uuid: cf080233-4a97-4d92-bb24-1232cc0bd073
title: "Platform Glossary & Ubiquitous Language"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-03T09:00:00Z"
updated_at: "2026-07-12T14:30:00Z"
summary: >
  The canonical lexicon of terms, architectural concepts, and protocols establishing
  the Ubiquitous Language of Alexandria.
audience: [public]
tags: [ "glossary", "reference", "vocabulary" ]
relations: []
---
# Platform Glossary & Ubiquitous Language

## Core System Terms

### Alexandria
The single, canonical repository serving as the "arctic vault" for all shared Go modules, API contracts, blueprints, and documentation assets in our platform ecosystem.

### Go Module
An independently versioned unit of Go source code rooted at a specific `go.mod` file within the Alexandria workspace directory structure (e.g. `go/retry`, `go/slog-gcp`).

### API Contract
A formal, declarative specification describing service interfaces and data exchange structures. Alexandria unifies these under standard Protocol Buffer schemas, managed via the **Buf CLI**.

---

## Architectural Terms

### OKF (Open Knowledge Format)
A Google Cloud specification that standardizes how organizational knowledge is packaged, stored, and exchanged. It defines a standard directory layout of plain text Markdown files initialized with highly structured YAML frontmatter.

### Directed Semantic Graph (DSG)
A queryable semantic network constructed from OKF documents. The graph is built by parsing the `relations` block of each document to establish typed edges (e.g., `depends_on`, `supersedes`) between document UUIDs.

### DWD (Domain-Wide Delegation)
A Google Workspace administrative feature allowing a Google Cloud Service Account to programmatically impersonate a designated Google Workspace user to access authorized Directory or Admin API scopes.

### Keyless Authentication
A secure authorization pattern that relies on Google Cloud Service Account OIDC credentials (such as Workload Identity) rather than static, unencrypted JSON credential key-files committed to disk.

---

## Engineering Standards

### Locality of Behavior (LoB)
An engineering principle dictating that the logic of a component and its immediate operational documentation should live in close proximity to maximize developer understanding and prevent documentation decay.

### SRE-Hardened
Code that is optimized for resilience under intense production loads: implementing lock-free fast-paths, zero-allocation serialization, keeping TCP connection pools hot, and containing automatic exponential retry backoffs.
