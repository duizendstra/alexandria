---
title: Alexandria OKF Profile
domain: reference
type: guide
diataxis_quadrant: reference
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: How Alexandria uses and extends the Open Knowledge Format (OKF) for its documentation vault.
uuid: 2766d6ff-0830-4010-a0b8-313da19f21ad
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "okf", "frontmatter", "reference" ]
relations: []
---

# Alexandria OKF Profile

Alexandria's `docs/` directory is an [OKF](https://okf.md) knowledge bundle — a
directory tree of markdown files with YAML frontmatter, as defined by the
[Open Knowledge Format v0.1 specification](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md).

This document describes how Alexandria applies and extends the OKF spec.

## Upstream Spec

OKF is an open, human- and agent-friendly format created by Google for
representing knowledge as markdown files. The core rules are minimal:

1. Every concept is a single `.md` file with YAML frontmatter.
2. `type` is the only required frontmatter field.
3. `index.md` and `log.md` are reserved filenames (directory listing and
   update history, respectively).

Full spec: [GoogleCloudPlatform/knowledge-catalog — SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)

## Alexandria Extensions

Alexandria extends the OKF frontmatter with the following fields. These are
not part of the upstream spec — they are conventions specific to this project.

### Frontmatter Schema

This is the canonical schema mandated by
[ADR-0002](../04-decisions/adr-0002-vault-centric-documentation.md) and
enforced by the OKF integrity lint in CI (`scripts/okf-lint.py`). Every
document in the vault declares all of these fields.

| Field | Required | Type | Description |
|---|---|---|---|
| `uuid` | ✅ | string | Alexandria-specific. Immutable RFC 4122 v4 identifier; unique across the vault. Relations point at these. |
| `type` | ✅ | string | OKF-standard. The kind of document (e.g., `index`, `guide`, `architecture_decision_record`). |
| `title` | ✅ | string | OKF-recommended. Human-readable display name. |
| `domain` | ✅ | string | Alexandria-specific. Which of the 8 domains this document belongs to. |
| `diataxis_quadrant` | ✅ | string | Alexandria-specific. Classifies the document per the [Diátaxis framework](https://diataxis.fr/). |
| `status` | ✅ | string | Alexandria-specific. Lifecycle state of the document. |
| `maturity` | ✅ | string | Alexandria-specific. Quality/completeness level. |
| `audience` | ✅ | list | Alexandria-specific. Who the document is for. |
| `owner` | ✅ | string | Alexandria-specific. GitHub handle of the responsible maintainer. |
| `summary` | ✅ | string | OKF-recommended (as `description`). One-line summary of the document; written as a dense digest to optimize semantic retrieval. |
| `created_at` | ✅ | string | Alexandria-specific. ISO 8601 UTC timestamp of first authoring; immutable. |
| `updated_at` | ✅ | string | Alexandria-specific. ISO 8601 UTC timestamp of the last substantive edit. |
| `tags` | ✅ | list | Alexandria-specific. Free-form lowercase topic tags for filtering and retrieval. |
| `relations` | ✅ | list | Alexandria-specific. Typed links to other vault documents; `[]` when none. See below. |

### Relations

Each relation is a map with exactly two keys, pointing at another vault
document's `uuid`:

```yaml
relations:
  - target_uuid: "7c10b0bc-5cb8-4eb4-b99b-efb829623dc1"
    rel_type: "depends_on"
```

`rel_type` is free-form but drawn from a small working set (`depends_on`,
`extends`, `supersedes`, `relates_to`). The lint verifies every
`target_uuid` resolves to a document in the vault.

### Field Values

#### `domain`

One of the 8 standard domains:

| Value | Folder | Verb |
|---|---|---|
| `governance` | `01-governance/` | GOVERN |
| `strategy` | `02-strategy/` | STRATEGIZE |
| `architecture` | `03-architecture/` | DESIGN |
| `decisions` | `04-decisions/` | DECIDE |
| `security` | `05-security/` | PROTECT |
| `operations` | `06-operations/` | RUN |
| `playbooks` | `07-playbooks/` | GUIDE |
| `reference` | `08-reference/` | LOOK UP |

#### `diataxis_quadrant`

One of the four [Diátaxis](https://diataxis.fr/) documentation types:

| Value | Purpose | Example |
|---|---|---|
| `tutorial` | Learning-oriented | Step-by-step first-module walkthrough |
| `how-to` | Task-oriented | "How to publish a Go module" |
| `reference` | Information-oriented | API docs, OKF spec |
| `explanation` | Understanding-oriented | ADRs, architecture rationale |

#### `status`

| Value | Meaning |
|---|---|
| `active` | Current and maintained |
| `draft` | Work in progress, not yet reviewed |
| `proposed` | Awaiting approval (used for ADRs) |
| `accepted` | Approved (used for ADRs) |
| `superseded` | Replaced by a newer document |
| `deprecated` | No longer relevant |

#### `maturity`

| Value | Meaning |
|---|---|
| `seed` | Placeholder or skeleton — minimal content |
| `draft` | Substantive content, not yet reviewed |
| `standard` | Reviewed and considered stable |

#### `audience`

A list of target audiences:

| Value | Meaning |
|---|---|
| `public` | External consumers and contributors |
| `internal` | Maintainers only |

### Type Values

Alexandria uses the following `type` values:

| Value | Used For |
|---|---|
| `index` | OKF-reserved directory listings (`index.md`) |
| `guide` | Narrative documentation, how-to guides, profiles |
| `architecture_decision_record` | ADRs in `04-decisions/` |

## The 8-Domain Convention

The 8-folder structure under `docs/` is an Alexandria convention, not part of
the OKF spec. OKF is intentionally flexible about directory organization.

Alexandria uses numbered prefixes (`01-`, `02-`, ...) to enforce consistent
ordering across tools and IDEs. Each domain folder contains an `index.md`
(the OKF-reserved directory listing) and concept documents.

## File Naming

| Convention | Source |
|---|---|
| `index.md` for directory listings | OKF spec (reserved filename) |
| `README.md` for GitHub-rendered entry points | GitHub convention |
| `adr-NNNN-slug.md` for ADRs | MADR convention |

The `docs/` root uses `README.md` (not `index.md`) because it serves as the
GitHub-rendered entry point to the vault. All subdirectories use `index.md`
per the OKF spec.
