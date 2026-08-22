---
uuid: a885b979-84f0-4537-954f-6285164e7208
title: "Don't Decorate Shared Adapters Used by Evidence-Producing Code"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T09:00:00Z"
updated_at: "2026-08-22T09:00:00Z"
summary: >
  Retry and error-wrapping helpers rewrite the error text they pass through;
  adding one to a shared adapter method silently changes any downstream
  string comparison, including verifier and audit-evidence output.
audience: [public]
tags: [ "retry", "error-handling", "shared-code", "composition", "lessons-learned" ]
relations: []
---

# Don't Decorate Shared Adapters Used by Evidence-Producing Code

A retry or error-wrapping helper is a decorator: it takes a function, calls
it, and on error returns a *new* error built from the original. That new
error's text is not the original's text — and code elsewhere may depend on
the original.

## Symptom

Some downstream code — a verifier, an audit log, a compliance report —
compares or parses an error string produced by a shared library call. After
a retry/wrapping helper is added to that shared call, the downstream
comparison starts failing, or the recorded evidence text silently changes,
even though the underlying failure is unchanged and even on a single,
non-retried failure.

## Cause

A generic retry wrapper commonly formats its own error regardless of retry
count, e.g. `operation failed after %d attempts: %w`. On the very first and
only attempt, `%d` is `1`, but the message still gained a new prefix. Any
caller that was comparing against, parsing, or logging the *original* error
text now sees the wrapped one instead. The helper was added to a shared
adapter method — one used by multiple callers — so every caller's error
shape changed at once, including the one whose exact wording was load-bearing.

## Rule

Never add retry, timeout, or error-wrapping behavior to a shared adapter
method that evidence-producing or verification code depends on for its
error text. Add the decoration at the *new* caller's composition root
instead — wrap the call site that needs retries, not the shared library
function everyone calls. If several callers genuinely need the same retry
policy, give them their own thin wrapper around the shared adapter rather
than modifying the adapter itself.

## How to Test for It

- Before merging any change to a shared adapter/client package, run
  `git diff <base>...HEAD -- <shared-package-path>` and read every line —
  a shared file should very rarely appear in a PR that also touches an
  unrelated caller.
- Grep the codebase for every place that compares, matches, or parses an
  error string returned by the adapter method being changed
  (`errors.Is`, `strings.Contains`, string-equality assertions in tests).
  Confirm each one still receives the exact text it expects.
- Add a test that calls the shared adapter method directly (no retry
  wrapper in the call path) and asserts its error text is unchanged by the
  presence of retry logic elsewhere in the codebase.
