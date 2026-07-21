---
title: Playbooks
domain: playbooks
type: index
diataxis_quadrant: how-to
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: How-to guides for adding modules, migrating from private packages, and publishing to pkg.go.dev.
uuid: f93b71c5-2fef-48e3-9219-714ad1543083
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "index", "playbooks" ]
relations: []
---

# 07 — Playbooks

This folder contains step-by-step how-to guides for common development and maintenance tasks.

## What Belongs Here

- **Adding a New Module** — Scaffolding a new Go module under `go/`, wiring CI, and creating initial documentation.
- **Migrating from Private Packages** — Steps to move an internal package into Alexandria as a public module.
- **Publishing to pkg.go.dev** — Tagging, versioning, and post-publish verification workflow.

## Contents

- [onboarding.md](onboarding.md) — Step-by-step developer learning playbook to set up a 100% declarative, local hermetic workspace using Nix in under 60 seconds.
- [cross-module-release.md](cross-module-release.md) — How to land and release a change spanning a module and its consumers: staged version pins, local verification with an uncommitted `go.work`, and path-prefixed tagging in dependency order.
- [golangci-resolutions.md](golangci-resolutions.md) — Cheat-sheet of recurring fixes for reaching 0 lint issues under the library and consumer profiles, including the pulumi `forcetypeassert` rewrite and the goconst-counts-tests trap.
