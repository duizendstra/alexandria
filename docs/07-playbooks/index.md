# 07 — Playbooks

This folder contains step-by-step how-to guides for common development and maintenance tasks.

## What Belongs Here

- **Adding a New Module** — Scaffolding a new Go module under `go/`, wiring CI, and creating initial documentation.
- **Migrating from Private Packages** — Steps to move an internal package into Alexandria as a public module.
- **Publishing to pkg.go.dev** — Tagging, versioning, and post-publish verification workflow.
- **Adopting Alexandria Modules** — De-vendoring duplicate packages, removing local replace directives, and adopting canonical platform modules.
- **Ascension & Consumption** — The two-directional protocol between Alexandria and a downstream consumer repository: depending by tag, and ascending a staged package into a shared module.

## Contents

* [Adding a New Module](adding-a-module.md) - How to scaffold a new Go module under go/, author it to the zero-rot package standard, wire it into CI (module index, Dependabot, coverage baseline), verify it locally, and release it with a path-prefixed tag.
* [Developer Onboarding Playbook](onboarding.md) - Step-by-step developer learning playbook for Alexandria, achieving a 60-second local development setup using Nix flakes and nix-direnv.
* [Cross-Module Release Playbook](cross-module-release.md) - How to land and release a change that spans an Alexandria module and the consumer modules that pin it: staging future-version pins, verifying locally with an uncommitted go.work, and tagging path-prefixed versions in dependency order after merge.
* [golangci-lint Resolutions Cheat-Sheet](golangci-resolutions.md) - Recurring fixes for getting a Go change to 0 issues under the Alexandria golangci profiles: the library vs consumer posture, the gocritic/nonamedreturns struct pattern, err113 sentinels, the pulumi forcetypeassert rewrite, and the goconst-counts-tests trap.
* [Adopting Alexandria Modules in Downstream Consumers](module-adoption.md) - How-to guide for downstream repositories adopting tagged Alexandria Go modules, eliminating vendored/duplicate packages, removing local replace directives, and integrating shared platform primitives.
* [Ascension & Consumption Protocol](ascension-and-consumption.md) - How Alexandria and a downstream consumer repository collaborate in both directions: consuming modules by tag behind blocking consumer contract tests, and ascending a generic package up from a consumer's staging lane through a read-only scaffold to a merged, tagged module.
