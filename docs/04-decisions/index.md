---
title: Decisions
domain: decisions
type: index
diataxis_quadrant: explanation
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: Architecture Decision Records (ADRs) in MADR format, documenting key technical choices and their rationale.
---

# 04 — Decisions

This folder contains Architecture Decision Records (ADRs) that capture significant technical choices, their context, and consequences.

All ADRs follow the [MADR](https://adr.github.io/madr/) (Markdown Any Decision Records) format.

## What Belongs Here

- **ADRs** — One file per decision, numbered sequentially (e.g., `adr-0001-use-slog.md`).
- **Superseded ADRs** — Kept for historical context, marked with `status: superseded`.

## ADR Template

Use the following template when creating a new ADR:

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

| ADR | Title | Status |
|---|---|---|
| [ADR-0001](adr-0001-monorepo-strategy.md) | Use a Multi-Module Monorepo | Accepted |
| [ADR-0002](adr-0002-vault-centric-documentation.md) | Vault-Centric Documentation Structure | Accepted |
