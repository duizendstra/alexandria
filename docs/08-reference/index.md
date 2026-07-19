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
uuid: f83b91ac-d6e1-488d-a4e5-86e7e8d4174c
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "index", "reference" ]
relations: []
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
| [`blueprints/`](../../blueprints/README.md) | Golden configuration templates: the `service/` Go Cloud Run ko build config, the `githooks/` Conventional Commits + quality-gate hook set, and the `golangci/` library/consumer lint profiles. |
