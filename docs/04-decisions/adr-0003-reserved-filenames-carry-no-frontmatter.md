---
uuid: c8549b5d-018c-494b-a03c-95bf76bcc208
title: "ADR-0003: Reserved OKF Filenames Carry No Frontmatter"
domain: "decisions"
type: "architecture_decision_record"
diataxis_quadrant: "explanation"
status: "accepted"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-23T07:35:03Z"
updated_at: "2026-08-23T07:35:03Z"
summary: >
  Amends ADR-0002 so index.md and log.md follow OKF §8 and §9 instead of the
  vault's concept-document frontmatter schema, closing an undisclosed
  deviation from the upstream specification.
audience: [public]
tags: [ "adr", "documentation", "okf" ]
relations:
  - target_uuid: "b4bc306c-9ba5-4eb8-b99b-efb829623dc1"
    rel_type: "amends"
---
# ADR-0003: Reserved OKF Filenames Carry No Frontmatter

## Status

Accepted

## Context

[ADR-0002](adr-0002-vault-centric-documentation.md) established `docs/` as an
[Open Knowledge Format](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)
bundle and, in the same breath, required *every* document in the vault to
declare the full Alexandria frontmatter schema. Those two rules do not agree.

OKF reserves two filenames at every level of the hierarchy (§3.1): `index.md`
for directory listings and `log.md` for update history. Neither is a concept
document. §8 states that index files contain no frontmatter, with one
exception — a bundle-root `index.md` may carry an `okf_version` key (§12) —
and §9 gives log files their own structure, entries grouped under ISO 8601
date headings.

Alexandria did the opposite. All eight domain `index.md` files carried the
same fourteen-field concept schema as every guide and ADR, including
`type: index`, a `uuid` other documents could point relations at, and
`created_at` / `updated_at` timestamps. `scripts/okf-lint.py` did not merely
permit this — it *required* it, applying one schema to every `.md` under
`docs/` without distinguishing reserved filenames. The vault claimed OKF
conformance in `docs/08-reference/okf-profile.md` while a lint gate enforced
the opposite of §8, and nothing disclosed the gap.

Two options were on the table:

1. **Disclose** — keep the frontmatter and record the deviation in the OKF
   profile as a deliberate Alexandria extension.
2. **Conform** — strip the frontmatter, teach the lint the reserved-filename
   rules, and drop the deviation.

Option 1 is cheap, but it trades away the property that makes the vault worth
formatting at all: an OKF consumer that meets §11 conformance can read the
bundle without knowing anything about Alexandria. A local extension that
inverts a spec rule is not an extension — every §8-aware consumer that opens
an index expecting a listing gets a metadata block instead.

The one thing option 1 protected was relation targets: a concept document may
point a `relations` entry at any `uuid` in the vault, and stripping index
frontmatter removes eight of them. In practice exactly one such relation
existed, from `07-playbooks/module-adoption.md` to its own folder index —
a pointer from a document to the listing that contains it, which carried no
information the file path did not already carry.

## Decision

Conform. This amends clause 2 (**Standard Metadata Schema**) and clause 3
(**Automated Schema Linters**) of ADR-0002; everything else in ADR-0002 stands.

1.  **Reserved filenames are not concept documents.** `index.md` and `log.md`,
    at any level of `docs/`, carry none of the Alexandria frontmatter schema.
    The schema in ADR-0002 clause 2 applies to concept documents — every `.md`
    that is not a reserved filename.
2.  **Indexes hold no frontmatter, with one exception.** The bundle-root
    `docs/index.md` declares `okf_version` and nothing else, making the spec
    version the vault targets machine-readable (§12).
3.  **The lint enforces both halves.** `scripts/okf-lint.py` checks concept
    documents against the full Alexandria schema exactly as before, and checks
    reserved filenames against §8 and §9 — no frontmatter in an index, a
    single `okf_version` key at the bundle root, listing entries grouped under
    headings, and ISO 8601 date headings in a log. `type: index` is removed
    from the type enum, because no document declares it any more.
4.  **Relations point at concept documents.** An index has no `uuid` to
    target. The single relation that pointed at one was removed.

## Consequences

### Easier

*   **Portable by construction** — the vault now satisfies OKF v0.2 §11
    conformance as written, so any §8-aware consumer reads the indexes as
    listings rather than tripping over a metadata block.
*   **Cheaper indexes** — an index is a directory listing again. Adding a
    document means adding one line, not minting a UUID and maintaining
    timestamps for a file that describes no concept.
*   **The claim matches the code** — `okf-profile.md` says the vault is OKF
    v0.2, `docs/index.md` declares it, and the lint enforces it.

### Harder

*   **Two rule sets to hold in mind** — an author now has to know that
    `index.md` and `log.md` are different, where before one schema covered
    everything. The lint reports which rule it applied, and the reserved names
    are the only exception.
*   **Indexes are no longer relation targets** — a document that wants to
    point at a group of documents has to point at them, or at a concept
    document that describes the group. Nothing in the vault needed this.
*   **Index metadata is gone** — the `owner`, `status`, and `updated_at` of an
    index are no longer recorded. They described the listing, not the
    knowledge, and git history covers the same ground.
