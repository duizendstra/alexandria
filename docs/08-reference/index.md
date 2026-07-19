---
title: Reference
domain: reference
type: index
diataxis_quadrant: reference
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: OKF specification and external API references.
---

# 08 — Reference

This folder contains look-up material — specifications, API references, and standards that other documents point to.

## What Belongs Here

- **OKF Specification** — The Open Knowledge Format standard used for all documentation in this repository.
- **External API References** — Links and notes on third-party APIs consumed by Alexandria modules.
- **Pointers to Non-Doc Assets** — Where to find the repository's shareable assets that live outside `docs/`.

## Contents

| Document | Description |
|---|---|
| [Alexandria OKF Profile](okf-profile.md) | How Alexandria uses and extends the Open Knowledge Format. |
| [Glossary](glossary.md) | The canonical lexicon of terms and Ubiquitous Language establishing architectural patterns. |

## Repository Assets Outside the Vault

| Asset | Description |
|---|---|
| [`skills/`](../../skills/README.md) | Antigravity AI skills shareable across workspaces (dialectical-review, diffract-review, ko-build, release-review); consumer repos inherit them via `skills.json`. |
| [`blueprints/`](../../blueprints/README.md) | Project scaffolding templates for bootstrapping new repositories (currently the `service/` Go Cloud Run blueprint with its golden ko build config). |
