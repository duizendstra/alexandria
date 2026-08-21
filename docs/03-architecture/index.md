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
uuid: 9af21e0d-99a7-4fae-97f9-53f3b8a5f746
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-08-21T11:35:00Z"
tags: [ "index", "architecture" ]
relations: []
---

# 03 — Architecture

This folder documents the technical design principles and structural patterns shared across all Alexandria modules.

## What Belongs Here

- **Package Design Patterns** — Conventions for public API surface, internal packages, and option functions.
- **Module Boundaries** — Rules for dependency isolation between modules.
- **Decorator Patterns** — How middleware, wrappers, and composable behaviors are structured.
- **Handler Chains** — Patterns for request/response pipelines and processing chains.
- **Agentic Platform Engineering** — Architectural guardrails and verification workflows for AI agents and human engineers.

## Contents

- [domain-driven-design-boundaries.md](domain-driven-design-boundaries.md) — Definitive standards for Domain-Driven Design (DDD) layer boundaries and dependency directions across all modules.
- [go-agentic-engineering.md](go-agentic-engineering.md) — Architectural principles for human-agent pair programming, verification-first development, and Go platform guardrails.
- [governance-domain-model.md](governance-domain-model.md) — The go/governance domain model (plan, scope, tiers, hierarchy, classification) and how the iac blueprints realize it on Google Cloud.
- [writing-enterprise-go-packages.md](writing-enterprise-go-packages.md) — Symbiotic doc.go and example_test.go patterns for compiler-validated zero-rot packages.
