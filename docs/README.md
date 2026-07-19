---
title: Alexandria Documentation Vault
domain: governance
type: guide
diataxis_quadrant: reference
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: Entry point to the Alexandria documentation vault — the arctic vault of shared engineering knowledge.
uuid: bfa87686-2f8c-48d3-8729-9a2288591d95
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "vault", "overview" ]
relations: []
---

# Alexandria — Documentation Vault

> The arctic vault. Shared knowledge for building cloud-native software.

## The 8-Folder Standard Domains

| Domain | Verb | Purpose | Index |
|---|---|---|---|
| **01-governance** | GOVERN | Licensing, contributing, changelog, and release policy. | [index](01-governance/index.md) |
| **02-strategy** | STRATEGIZE | Vision, roadmap, and long-term direction. | [index](02-strategy/index.md) |
| **03-architecture** | DESIGN | Package design, module boundaries, and patterns. | [index](03-architecture/index.md) |
| **04-decisions** | DECIDE | Architecture Decision Records (ADRs) using MADR. | [index](04-decisions/index.md) |
| **05-security** | PROTECT | Dependency policy, vulnerability handling. | [index](05-security/index.md) |
| **06-operations** | RUN | CI/CD, release automation, and maintenance. | [index](06-operations/index.md) |
| **07-playbooks** | GUIDE | How-to guides, migration recipes, troubleshooting. | [index](07-playbooks/index.md) |
| **08-reference** | LOOK UP | API references, OKF spec, and external links. | [index](08-reference/index.md) |

## Conventions

- **[OKF](https://okf.md)** — This vault is an [Open Knowledge Format](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) bundle. See the [Alexandria OKF Profile](08-reference/okf-profile.md) for how we extend the spec.
- **Numbered prefixes** enforce consistent directory ordering across tools and IDEs.
- **`index.md`** in each directory maps folder-local documents (OKF reserved filename).
- **YAML frontmatter** on every document ensures machine-readability for agentic consumption.
- **ADRs** follow the MADR format — one decision per file in `04-decisions/`.
