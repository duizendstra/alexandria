---
uuid: b5ca5a05-41c7-49cb-8cb6-06ea4258c90b
title: "Cross-Module Release Playbook"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "draft"
owner: "@duizendstra"
created_at: "2026-07-21T06:05:55Z"
updated_at: "2026-07-21T06:05:55Z"
summary: >
  How to land and release a change that spans an Alexandria module and the
  consumer modules that pin it: staging future-version pins, verifying locally
  with an uncommitted go.work, and tagging path-prefixed versions in dependency
  order after merge.
audience: [public]
tags: [ "release", "versioning", "multi-module", "go", "playbook" ]
relations:
  - target_uuid: "599a1bf7-9a16-4369-8574-4126261e7578"
    rel_type: "depends_on"
  - target_uuid: "ef14ea7c-5f4b-40dd-a407-276836e3fd11"
    rel_type: "relates_to"
---

# Cross-Module Release Playbook

Alexandria is a multi-module monorepo: each module under `go/` is versioned
independently with path-prefixed tags (`go/<module>/vX.Y.Z`) and pins its
Alexandria dependencies by version, not by `replace`. A change that spans a
module **and** the consumers that pin it therefore cannot land atomically — the
consumer can only pin a version that exists once the producer is tagged.

This playbook is the manual process for those releases. Single-module changes
that break nothing downstream do not need it: land the PR and tag the one
module.

## When to use this

Use it whenever a single logical change touches a module and one or more of its
consumers — most often a **breaking change** in a lower module that composite
modules must adopt. Example dependency direction:

```
leaf  (e.g. governance) → platform → composite (e.g. iac/governance)
```

The module graph is a clean DAG: releases flow **leaf → composite**.

## Mental model

- **Producers are tagged before consumers can pin them.** A consumer `go.mod`
  that references `go/<producer>/vX.Y.Z` will not build until that tag is
  pushed. This ordering is intrinsic, not a workaround.
- **A cross-module release is staged, then sequenced.** You stage the intended
  pins in one wave of PRs, then, after merge, create the tags in dependency
  order so each consumer's pin resolves.

## Procedure

### 1. Stage the future-version pins

In the same change wave, edit each consumer's `go.mod` to pin the **version the
producer is about to be tagged with** (not the current one). Choose that version
now so the whole wave is internally consistent.

> **Expected:** CI for the staged-pin consumer modules goes **red** until the
> producer tags exist — the pinned version is unresolvable. State this
> explicitly in the PR body so reviewers do not read it as a real failure.

Before choosing numbers, list existing tags — parallel workstreams move them
fast, and a number you assumed was free may already be taken:

```bash
git tag -l 'go/<module>/*'
```

### 2. Verify locally before merge

Because the pinned tags do not exist yet, prove the wave builds together with an
**uncommitted** `go.work` that redirects the staged modules to your local tree:

```go
// go.work — DO NOT COMMIT
go 1.26

use (
    ./go/<producer>
    ./go/<consumer>
)

// A plain `use` is not enough: the module graph still resolves the pinned
// version's go.mod from the proxy. Add replace directives so the staged
// version resolves to local source.
replace github.com/duizendstra/alexandria/go/<producer> => ./go/<producer>
```

Run the affected modules' gates with `GOWORK=off` for the real per-module CI
picture, and with the work file for the integrated check. Delete the `go.work`
before pushing (it is git-ignored, but confirm it is not staged).

### 3. Tag in dependency order after merge

Once the wave is merged to `main`, create the tags **leaf first**, so each
consumer's pin resolves before that consumer is tagged:

```bash
git tag go/<producer>/vX.Y.Z
git push origin go/<producer>/vX.Y.Z
```

For each staged-pin consumer, refresh its module graph against the now-real tag
and open a small follow-up `go mod tidy` PR, then tag the consumer:

```bash
# GOPRIVATE avoids proxy / checksum-db lag on a tag pushed seconds ago.
GOPRIVATE=github.com/duizendstra/alexandria go mod tidy
```

Repeat up the DAG until every module in the wave is tagged.

### 4. Keep the module index honest

Update the version column in the root `README.md` module index. The CI parity
gate checks that every module is *present* in the table, not that its version is
current — so the version is yours to keep accurate.

## Gotchas

- **Retargeting a PR does not retrigger CI.** Push an empty commit to force a
  fresh run:
  ```bash
  git commit --allow-empty -m "ci: retrigger"
  ```
- **The `main` ruleset requires branches to be up to date.** On a fast-moving
  `main`, several releasable PRs will race for the same window. Coordinate merge
  windows rather than fighting repeated "out of date" rejections.
- **Fresh tags lag the proxy.** If `go mod tidy` cannot find a tag you just
  pushed, retry with `GOPRIVATE` set (as above) before assuming the tag is
  wrong.

## Related

- [ADR-0001: Use a Multi-Module Monorepo](../04-decisions/adr-0001-monorepo-strategy.md)
  — why modules are versioned independently.
- [Declarative CI/CD Pipelines & Release Automation](../06-operations/declarative-ci-cd-pipelines.md)
  — the pipeline this manual process runs on top of.
- `CONTRIBUTING.md` — the versioning and Conventional Commits conventions this
  process assumes.
