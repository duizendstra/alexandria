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
updated_at: "2026-08-23T07:35:03Z"
tags: [ "okf", "frontmatter", "reference" ]
relations: []
---

# Alexandria OKF Profile

Alexandria's `docs/` directory is an [OKF](https://okf.md) knowledge bundle — a
directory tree of markdown files with YAML frontmatter, as defined by the
[Open Knowledge Format v0.2 specification](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md).

This document describes how Alexandria applies and extends the OKF spec.

## Upstream Spec

OKF is an open, human- and agent-friendly format created by Google for
representing knowledge as markdown files. The core rules are minimal:

1. Every concept is a single `.md` file with YAML frontmatter.
2. `type` is the only required frontmatter field.
3. `index.md` and `log.md` are reserved filenames (directory listing and
   update history, respectively) and are **not** concept documents.

Full spec: [GoogleCloudPlatform/open-knowledge-format — SPEC.md](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)

### Conformance

OKF v0.2 §11 defines a conformant bundle with three conditions. Each is a
gate in `scripts/okf-lint.py`, so the vault cannot drift out of conformance
without failing CI:

| §11 condition | Where it is enforced |
|---|---|
| Every non-reserved `.md` has a parseable YAML frontmatter block | Frontmatter parse; a missing or unterminated block is a finding. |
| Every frontmatter block has a non-empty `type` | `type` is in the required-field list and enum-checked. |
| Every reserved filename follows §8 (`index.md`) and §9 (`log.md`) | The reserved-filename checks below. |

Alexandria then layers its own required fields on top. That is an extension,
not a deviation: §4.1 lets producers add keys and §11 forbids consumers from
rejecting a document for carrying unrecognized ones.

## Alexandria Extensions

Alexandria extends the OKF frontmatter with the following fields. These are
not part of the upstream spec — they are conventions specific to this project.

### Frontmatter Schema

This is the canonical schema mandated by
[ADR-0002](../04-decisions/adr-0002-vault-centric-documentation.md) and
enforced by the OKF integrity lint in CI (`scripts/okf-lint.py`). Every
**concept document** declares all of these fields — that is, every `.md` in
the vault except the reserved filenames `index.md` and `log.md`, which carry
no schema at all ([ADR-0003](../04-decisions/adr-0003-reserved-filenames-carry-no-frontmatter.md)).

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
`extends`, `supersedes`, `amends`, `relates_to`). The lint verifies every
`target_uuid` resolves to a document in the vault.

Only concept documents have a `uuid`, so only concept documents can be
relation targets. To point at a group of documents, point at the documents.

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

Upstream OKF v0.2 also defines a `status` key (`draft | stable | deprecated`);
Alexandria's `status` predates it and uses its own lifecycle enum below. Within
this vault the Alexandria values are authoritative.

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
| `guide` | Narrative documentation, how-to guides, profiles |
| `architecture_decision_record` | ADRs in `04-decisions/` |

There is no `index` type: an `index.md` has no frontmatter to declare one.

## The 8-Domain Convention

The 8-folder structure under `docs/` is an Alexandria convention, not part of
the OKF spec. OKF is intentionally flexible about directory organization.

Alexandria uses numbered prefixes (`01-`, `02-`, ...) to enforce consistent
ordering across tools and IDEs. Each domain folder contains an `index.md`
(the OKF-reserved directory listing) and concept documents.

## Reserved Filenames

`index.md` and `log.md` mean something at every level of the hierarchy
(OKF §3.1). They are directory apparatus, not knowledge, so they carry none
of the frontmatter schema above — see
[ADR-0003](../04-decisions/adr-0003-reserved-filenames-carry-no-frontmatter.md)
for why the vault used to, and no longer does.

### `index.md` (§8)

An index enumerates its directory's contents so a reader — human or agent —
can see what is available before opening anything. Its body is one or more
sections of listing entries under headings:

```markdown
# Section Heading

* [Title](relative-path.md) - the linked document's summary
* [Subdirectory](subdir/) - what the subdirectory holds
```

Entries reuse the linked concept's `summary` as their description, which is
what §8 asks for and keeps the two from drifting apart.

An index holds **no frontmatter**, with exactly one exception: the
bundle-root `docs/index.md` declares the spec version the vault targets, and
nothing else (§12):

```yaml
---
okf_version: "0.2"
---
```

### `log.md` (§9)

No log file exists in the vault yet. When one is added, its entries group
under ISO 8601 `YYYY-MM-DD` headings, newest first — the lint rejects any
other section heading in a `log.md`.

### What the lint checks

| Rule | Spec |
|---|---|
| A non-root `index.md` carries no frontmatter | §8 |
| `docs/index.md` exists and declares `okf_version`, and no other key | §8, §12 |
| An index body has headings and at least one linked entry | §8 |
| Every `##` section heading in a `log.md` is a `YYYY-MM-DD` date | §9 |

## File Naming

| Convention | Source |
|---|---|
| `index.md` for directory listings | OKF spec (reserved filename) |
| `log.md` for update history | OKF spec (reserved filename) |
| `README.md` for GitHub-rendered entry points | GitHub convention |
| `adr-NNNN-slug.md` for ADRs | MADR convention |

The `docs/` root carries both, for two different readers. `index.md` is the
OKF bundle-root index — the listing a spec-aware consumer looks for, and the
one place `okf_version` is declared. `README.md` is the page GitHub renders
when someone browses to `docs/`; it is an ordinary concept document, and it
carries the orientation a newcomer needs (the 8-folder standard and the
conventions the vault follows) rather than a second copy of the listing.
