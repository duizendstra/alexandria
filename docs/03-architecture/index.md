---
title: Architecture
domain: architecture
type: index
diataxis_quadrant: explanation
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: Package design patterns, module boundaries, decorator patterns, and handler chains used across Alexandria modules.
---

# 03 — Architecture

This folder documents the technical design principles and structural patterns shared across all Alexandria modules.

## What Belongs Here

- **Package Design Patterns** — Conventions for public API surface, internal packages, and option functions.
- **Module Boundaries** — Rules for dependency isolation between modules.
- **Decorator Patterns** — How middleware, wrappers, and composable behaviors are structured.
- **Handler Chains** — Patterns for request/response pipelines and processing chains.

## Contents

- [domain-driven-design-boundaries.md](domain-driven-design-boundaries.md) — Definitive standards for Domain-Driven Design (DDD) layer boundaries and dependency directions across all modules.
- [governance-domain-model.md](governance-domain-model.md) — The go/governance domain model (plan, scope, tiers, hierarchy, classification) and how the iac blueprints realize it on Google Cloud.
