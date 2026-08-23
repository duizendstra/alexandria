---
name: alexandria-consumer
description: Follow the Alexandria ascension & consumption protocol when working in a repository that depends on Alexandria modules or stages packages for ascension. Activate when adopting or bumping an Alexandria tag, when staging a package in lib/go for ascension, or when filing an advisory or ascension issue.
---

# Alexandria Consumer Skill

This skill teaches the protocol between Alexandria and a downstream consumer
repository so a coordinator agent follows it without a bespoke brief. The
authoritative page is `docs/07-playbooks/ascension-and-consumption.md` in
Alexandria; this skill is the operational form of it.

## When to Activate

- Adopting an Alexandria module, or bumping a pinned tag
- Staging a generic package in the consumer's `lib/go/<path>` lane
- Implementing against a scaffold Alexandria has merged
- Filing a suggestion towards a downstream repository, or an ascension request

## Hard rules

1. **Depend by tag.** Never a `replace` directive, never a branch.
2. **Alexandria is read-only towards the consumer repository.** Working in
   Alexandria, read the staged package; never write into the other repository.
3. **Alexandria is public.** No consumer vocabulary, identifiers, domain nouns,
   repository names, or internal issue numbers reach it — not in code, not in
   docs, not in commit messages, not in issue or pull request titles and bodies,
   and not in label names.
4. **Never two live copies.** The staged copy is deleted in the same change that
   swaps the consumer to the tag.

## Consuming a module

1. Add or bump the dependency by tag.
2. For every promise relied on, keep a **blocking consumer contract test** in
   the consumer's own suite, stated in the consumer's terms.
3. Adopt the bump only when those tests are green. If one goes red, raise it
   with the module owner — do not silently pin backwards.

## Ascending a package

Check the entry bar before proposing anything. The package must be:

- **Generic** — no consumer-specific vocabulary, no internal imports
- **Module-shaped** — `doc.go`, `README.md`, runnable `example_test.go`, tests
- **Proven** — exactly one real consumer, not a speculative API

Then:

1. Stage it in the consumer's `lib/go/<path>` lane.
2. File an ascension issue in Alexandria (see labels below).
3. Alexandria scaffolds the module: API stubs, the contract in doc comments,
   test scaffolds, test infra. It does not carry the implementation.
4. The consumer team implements against the scaffold.
5. The maintainer merges **and tags in the same session**.
6. The consumer swaps its import to the tag and deletes the staged copy.

## Filing issues across repositories

| Direction | Label |
|---|---|
| Alexandria towards a consumer | `advisory` |
| A consumer towards Alexandria | `ascension` |

The labels are generic on purpose. **Name the originating or target repository
in the issue body, never in the label** — label names appear on public pages and
in every issue view, so a repository name in a label publishes the relationship.
Mirror the issue with a comment on the originating repository's tracking issue.

## Before opening a pull request in Alexandria

A reviewer agent pre-reviews against a fixed checklist and the maintainer
merges. Expect to be checked on: no consumer specifics; module shape; a coverage
baseline entry; the module's consumers and promises section; a CHANGELOG entry;
Conventional Commits; and a proposed tag.

## Scars flow up

When a lesson is learned the hard way downstream, propose the generic shape of
it as a short playbook or reference page in Alexandria. The incident stays
downstream; only the reusable shape ascends.
