---
uuid: 89efab66-0a49-4237-8e21-ddde2ea4686d
title: "Ascension & Consumption Protocol"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "draft"
owner: "@duizendstra"
created_at: "2026-08-23T09:18:49Z"
updated_at: "2026-08-23T09:18:49Z"
summary: >
  How Alexandria and a downstream consumer repository collaborate in both
  directions: consuming modules by tag behind blocking consumer contract tests,
  and ascending a generic package up from a consumer's staging lane through a
  read-only scaffold to a merged, tagged module.
audience: [public]
tags: [ "protocol", "ascension", "consumers", "versioning", "playbook" ]
relations:
  - target_uuid: "cd2acade-1fc9-4423-896b-07681165d1e1"
    rel_type: "relates_to"
  - target_uuid: "8cdf6842-b989-4ec1-8bf5-5666d8ab962e"
    rel_type: "relates_to"
---

# Ascension & Consumption Protocol

Code moves between Alexandria and a downstream consumer repository in two
directions, and each direction has its own rules.

- **Consumption** is Alexandria to the consumer: the consumer depends on tagged
  modules.
- **Ascension** is the consumer to Alexandria: a package that has proven itself
  downstream becomes a shared module here.

This page is the single source for both. It assumes the reader is an agent or
engineer working in either repository.

## Consumption

A consumer depends on Alexandria **by tag** — never by a local `replace`
directive, and never by branch. See
[Adopting Alexandria Modules in Downstream Consumers](module-adoption.md) for
the mechanics and [Cross-Module Release Playbook](cross-module-release.md) for
changes that span several modules.

For every promise it actually relies on, the consumer keeps a **blocking
consumer contract test** in its own suite. The test states the promise in the
consumer's terms, so that breaking it fails the consumer's build rather than
being discovered in production.

A tag bump is **adopted when those tests are green**. A red contract test is a
negotiation with the module owner, not a reason to pin backwards silently.

## Ascension

Ascension has four stages. The rule that shapes all of them: **Alexandria is
read-only towards the consumer repository.** It reads a staged package, and it
never writes into that repository.

### 1. Stage

The consumer stages the candidate in its `lib/go/<path>` lane. To be eligible
it must already be:

- **Generic** — no vocabulary, identifiers, or domain nouns specific to the
  consumer, and no imports from the consumer's internal packages.
- **Module-shaped** — `doc.go`, a `README.md`, a runnable `example_test.go`,
  and tests.
- **Proven** — exactly one real consumer using it, not a speculative API.

A package that fails any of these stays downstream until it does not.

### 2. Scaffold

Alexandria scaffolds the module from the staged package: the API stubs, the
contract written into doc comments, test scaffolds, and the test infrastructure.
The scaffold defines the shape and the promises; it does not carry the
implementation.

### 3. Implement

The consumer team implements against the scaffold. The contract in the doc
comments is what they are implementing to, so a disagreement surfaces as a
change to the contract rather than as a silent divergence.

### 4. Merge, tag, adopt

The maintainer merges and **cuts the tag in the same session** — an ascended
module that is merged but untagged cannot be consumed, and the downstream copy
cannot be removed until it can. The consumer then swaps its import to the tag
and **deletes its staged copy**. Two live copies is the failure state this
protocol exists to prevent.

## Advisories and ascension requests

Cross-repository suggestions travel as issues with a generic label:

| Direction | Label | Meaning |
|---|---|---|
| Alexandria to a consumer | `advisory` | A suggestion for a downstream repository. |
| A consumer to Alexandria | `ascension` | A request to ascend a staged package. |

**The labels are deliberately generic.** This repository is public, so the
originating or target repository is named **in the issue body, never in a
label** — label names are listed on public pages and in every issue view, and a
repository name in a label would publish the relationship. Mirror the issue with
a comment on the originating repository's own tracking issue so the two sides
stay linked.

## Review gate

Every Alexandria pull request is pre-reviewed by a reviewer agent against a
fixed checklist before the maintainer merges. The checklist covers: no consumer
specifics, module shape, a coverage baseline entry, the module's consumers and
promises section, the CHANGELOG entry, Conventional Commits, and a proposed tag.

The maintainer merges; the reviewer never does.

## Scars flow up

A lesson learned the hard way in a consumer repository becomes a short playbook
or reference page here, with the specifics stripped. The generic shape of the
failure is the reusable part; the incident that produced it stays downstream.
