---
uuid: 1aa57918-854b-4a0e-a95f-b1c328943779
title: "Governance Bounded Context: Domain Model"
domain: "architecture"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "draft"
owner: "@duizendstra"
created_at: "2026-07-19T12:00:00Z"
updated_at: "2026-07-19T12:00:00Z"
summary: >
  Describes the go/governance domain model — plan, scope, tiers, hierarchy,
  classification, and exports — and how the iac/governance blueprint and
  iac/pulumi/gcpinfra adapters realize it on Google Cloud.
audience: [public]
tags: [ "architecture", "governance", "ddd", "domain-model" ]
relations:
  - target_uuid: "ee14bc6c-1349-411a-8bb4-f58c70a84e20"
    rel_type: "extends"
---
# Governance Bounded Context: Domain Model

## Purpose

The Governance bounded context answers: *how are cloud resources organized and
controlled?* — what folders exist, what environments are available, how
resources are classified, and who pays. Its domain model lives in
[`go/governance`](../../go/governance/README.md), a pure-Go module with zero
external dependencies (v0.1.0 — the first module whose API shape has been
validated by a real consumer).

## Model

```
plan.Plan                    Desired governance state (pure data) — the port
scope.Scope                  Organization or Folder — determines capabilities
hierarchy.Config             Folder tree definition
classification.Dimension     Tag key / label axis
exports                      Values published for downstream bounded contexts
```

Two orthogonal axes shape every `Plan`:

*   **Scope** — *where* governance operates: a full GCP Organization or a
    sub-org Folder. Scope constrains which capabilities are available (e.g.,
    tag keys and billing exports are org-level resources).
*   **Tier** — *what* features are active, following Google Cloud naming:
    Starter (single root folder), Standard (environment folders), Enterprise
    (full tree, classification dimensions, billing/org exports; requires
    Organization scope).

Moving up a tier is additive and safe; moving down is destructive (resources
are deleted). See the module README for the full tier-by-scope matrix.

## Realization (Ports & Adapters)

The `Plan` is the port. Infrastructure realization follows the standard
[DDD boundary rules](domain-driven-design-boundaries.md):

*   [`go/iac/governance`](../../go/iac/governance/README.md) — the blueprint: a
    configuration-driven Pulumi program that reads config, builds and validates
    a domain `Plan`, then delegates to platform adapters and exports results
    for downstream bounded contexts.
*   [`go/iac/pulumi/gcpinfra`](../../go/iac/pulumi/gcpinfra/README.md) — thin,
    opinionated Pulumi adapter packages (folders, tag keys) that validate input
    via the domain package before creating GCP resources with deletion
    protection enabled.

The dependency direction is strictly one-way: `gcpinfra` and the blueprint
depend on `governance`; the domain module knows nothing about Pulumi or GCP
client libraries.
