# 04 — Decisions

This folder contains Architecture Decision Records (ADRs) that capture significant technical choices, their context, and consequences.

All ADRs follow the [MADR](https://adr.github.io/madr/) (Markdown Any Decision Records) format.

## What Belongs Here

- **ADRs** — One file per decision, numbered sequentially (e.g., `adr-0001-use-slog.md`).
- **Superseded ADRs** — Kept for historical context, marked with `status: superseded`.

## ADR Template

Use the following template when creating a new ADR. The frontmatter block
carries every field the [OKF profile](../08-reference/okf-profile.md) requires
of a concept document, so a file started from it passes `scripts/okf-lint.py`
once the placeholders are filled in.

```markdown
---
title: "ADR-NNNN: Short Decision Title"
domain: decisions
type: architecture_decision_record
diataxis_quadrant: explanation
status: proposed
maturity: seed
audience: [public]
owner: "@duizendstra"
summary: One-line summary of the decision.
uuid: <a fresh RFC 4122 v4 uuid>
created_at: "YYYY-MM-DDTHH:MM:SSZ"
updated_at: "YYYY-MM-DDTHH:MM:SSZ"
tags: [ "adr" ]
relations: []
---

# ADR-NNNN: Short Decision Title

## Status

Proposed

## Context

What is the issue that we are seeing that is motivating this decision or change?

## Decision

What is the change that we are proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?
```

## Contents

* [ADR-0001: Use a Multi-Module Monorepo](adr-0001-monorepo-strategy.md) - All shared Go modules, contracts, documentation, and tooling live in a single repository with independent versioning. *(Accepted)*
* [ADR-0002: Vault-Centric Documentation Structure](adr-0002-vault-centric-documentation.md) - Establishes a unified, OKF-compliant documentation vault to organize architectural, governance, and operational knowledge. *(Accepted)*
* [ADR-0003: Reserved OKF Filenames Carry No Frontmatter](adr-0003-reserved-filenames-carry-no-frontmatter.md) - Amends ADR-0002 so `index.md` and `log.md` follow OKF §8 and §9 instead of the vault's concept-document frontmatter schema. *(Accepted)*
