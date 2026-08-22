---
uuid: 9e2e6506-4501-458a-9f3a-16ecfbf397b4
title: "Concurrency-Safe Lazy Initialization: Mutex vs. sync.Once"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T09:00:00Z"
updated_at: "2026-08-22T09:00:00Z"
summary: >
  sync.Once guarantees its function runs exactly once, not exactly once on
  success — a first-call failure during lazy initialization of a shared
  client or repository is permanent for the life of the process unless a
  retryable mutex-based pattern is used instead.
audience: [public]
tags: [ "concurrency", "go", "sync", "initialization", "lessons-learned" ]
relations: []
---

# Concurrency-Safe Lazy Initialization: Mutex vs. sync.Once

Lazily initializing a shared client or repository under concurrent access
needs a guard so only one caller does the expensive setup. `sync.Once` is
the idiomatic Go tool for "run this exactly once" — but "exactly once"
includes a failed attempt, which is rarely what a lazy-init caller wants.

## Symptom

A shared repository or client is built lazily on first use, guarded by
`sync.Once`. In production, the very first initialization attempt fails
(a transient network error, a not-yet-ready dependency). Every subsequent
call — including ones made long after the transient condition has
cleared — returns the same failure (or a zero-value/uninitialized object)
for the remaining lifetime of the process, with no further attempt to
initialize.

## Cause

`sync.Once.Do(f)` guarantees `f` is invoked exactly once per `Once`
instance, regardless of whether `f` returns an error or panics. Once
`Do` has been called, the `Once` is permanently "done," and no later call
to `Do` will invoke `f` again — the poisoned-failure trap. If `f` sets up
a shared client and fails partway through, the `Once` still considers
itself fired, and there is no built-in way to make it try again.

## Rule

For lazy initialization that can fail (essentially anything that talks to
the network, disk, or another service on first use), do not use
`sync.Once`. Use a plain `sync.Mutex` guarding an explicit state — a
cached instance plus a cached error, or a small state enum
(uninitialized / ready / failed) — so that:

- concurrent callers during the *same* initialization attempt block and
  share its result (same property `sync.Once` gives you), and
- a *failed* attempt does not poison future callers — the next caller
  (or the next call after backoff) retries initialization from scratch.

Reserve `sync.Once` for initializers that are pure computation and cannot
meaningfully fail (e.g. compiling a regexp literal, building a static
lookup table).

## How to Test for It

- Write a unit test that injects a failure into the initializer's first
  invocation only (a call counter or a fake dependency that fails once)
  and asserts that a second, later call succeeds and returns a working
  instance. A `sync.Once`-based implementation fails this test: the second
  call still returns the cached failure/zero value.
- Grep the codebase for `sync.Once` guarding any function whose body
  performs I/O (network calls, file reads, RPC client construction) —
  each hit is a candidate for this trap and should be reviewed.
- Under a race detector, run many goroutines through the lazy-init path
  concurrently during a simulated first-attempt failure, and confirm they
  all observe the same retry outcome rather than a mix of stale poisoned
  errors and freshly initialized instances.
