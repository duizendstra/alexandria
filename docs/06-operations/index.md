---
title: Operations
domain: operations
type: index
diataxis_quadrant: explanation
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: CI/CD pipeline design, release-please automation, Dependabot configuration, and tagging conventions.
uuid: 3fbc550a-2bed-4f5a-bb7a-d853e9e67cf1
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "index", "operations" ]
relations: []
---

# 06 — Operations

This folder documents the automated pipelines, release tooling, and operational procedures that keep Alexandria healthy.

## What Belongs Here

- **CI/CD Pipeline Design** — GitHub Actions workflows for linting, testing, and publishing.
- **Release-Please Automation** — Configuration and conventions for automated changelog and version bumps.
- **Dependabot Config** — Dependency update schedules and grouping rules.
- **Tagging Conventions** — Module-scoped tag format (e.g., `go/slog-gcp/v0.1.0`).

## Contents

- [declarative-ci-cd-pipelines.md](declarative-ci-cd-pipelines.md) — Multi-module monorepos release strategies, semantic tagging, and automated regression metrics checks.
- [disaster-recovery-and-rollback.md](disaster-recovery-and-rollback.md) — Step-by-step operational instructions for hot-fixing, rolling back faulty releases, retracting Go modules, and recovering from pipeline incidents.

