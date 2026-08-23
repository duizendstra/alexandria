---
uuid: 71ad83dd-8fff-40b6-a6db-24bb0d9192c8
title: "Pull Request Review Checklist"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "draft"
owner: "@duizendstra"
created_at: "2026-08-23T09:36:53Z"
updated_at: "2026-08-23T09:36:53Z"
summary: >
  The fixed checklist a reviewer agent works through on every Alexandria pull
  request before the maintainer merges: no consumer specifics, module shape,
  coverage baseline, the consumers and promises section, the CHANGELOG entry,
  Conventional Commits, and a proposed tag.
audience: [public]
tags: [ "review", "process", "quality-gate", "go", "playbook" ]
relations:
  - target_uuid: "8cdf6842-b989-4ec1-8bf5-5666d8ab962e"
    rel_type: "relates_to"
  - target_uuid: "b5ca5a05-41c7-49cb-8cb6-06ea4258c90b"
    rel_type: "relates_to"
---

# Pull Request Review Checklist

Every pull request is worked through against this list before the maintainer
merges. The list is fixed on purpose: a reviewer that decides each time what
to look at is a reviewer that misses the same thing twice.

An item that does not apply is called out as not applicable, not skipped
silently. A reviewer that cannot tell whether an item passes reports it as
unresolved rather than assuming.

## 1. No consumer specifics

Alexandria is public. Nothing that identifies a downstream repository, its
organisation, its products, its environments, or its internal issue numbers
belongs anywhere in the change — not in code, fixtures, documentation, commit
messages, branch names, or the pull request title and body.

Consumers appear as **archetypes**, never as names: "Pulumi composition roots
that wire bounded contexts together", not the repository that happens to be
one. Test fixtures use obviously invented values; a realistic-looking
identifier is the one that survives review and ships.

Grep the added lines, not the whole file — context lines from an untouched
file are not this change's problem. Derive the terms to grep for from the
actual downstream vocabulary rather than guessing a list, and run a control
term that is known to be present, so a scan that matches nothing is
distinguishable from a scan that cannot match.

## 2. Module shape

A module carries a package `doc.go`, a `README.md`, and at least one
`example_test.go` exercising the documented entry point. A change that adds
a module without all three is incomplete; a change that adds an exported
entry point leaves the examples still representative.

## 3. Coverage baseline

A module touched by the change has its entry in
`.github/coverage-baselines.json`, and the number moves up or stays put.
The ratchet never loosens to make a change fit.

## 4. Consumers and promises

The module's **Consumers & Load-Bearing Promises** section reflects the change.
A promise listed there is a compatibility commitment: breaking one is a major
version bump, and the bump is proposed in the same pull request rather than
discovered later by whoever pinned it.

A new behaviour that consumers will reasonably rely on is added to the list
while it is still cheap to phrase. A promise that no test pins is not a
promise — it is a hope, and it is either given a test or dropped.

## 5. CHANGELOG

A notable change has a bullet under `## [Unreleased]` in the root
`CHANGELOG.md`, written for someone deciding whether to upgrade rather than
for someone reading the diff.

## 6. Conventional Commits

The commit subject is scoped to the module — `fix(filelock): …`,
`docs(okf): …` — and a breaking change is marked with `!`. The scope
matches the directory actually touched.

## 7. Proposed tag

The pull request states the path-prefixed tag it expects to produce —
`go/<module>/vX.Y.Z` — and the version reflects what section 4 concluded.
Tags are cut post-merge, in dependency order.

## Outcome

The review ends in one of three states, stated explicitly:

- **Approve** — every item passes.
- **Approve with nits** — everything load-bearing passes; the nits are listed
  so the author can take or leave them.
- **Request changes** — at least one item fails, named with the item number.

Section 1 is never a nit.
