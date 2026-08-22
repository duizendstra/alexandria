---
uuid: 9b91ebe3-cd82-4e3e-932f-a7356690a2a6
title: "Cross-Process Coordination on a Shared Remote Resource"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T09:00:00Z"
updated_at: "2026-08-22T09:00:00Z"
summary: >
  A per-process mutex does not protect a remote resource that several
  processes (or subprocess fan-out) mutate concurrently; use an advisory
  file lock around the mutating window and treat the remote API's conflict
  response as transient.
audience: [public]
tags: [ "concurrency", "locking", "distributed-systems", "retry", "lessons-learned" ]
relations: []
---

# Cross-Process Coordination on a Shared Remote Resource

An in-process mutex only ever coordinates goroutines/threads inside the
process that created it. When the actual contention is on a remote
resource — a shared drive's membership list, a shared record, a shared
bucket prefix — and more than one process (or a fan-out of subprocesses)
can touch that resource at the same moment, the mutex protects nothing.

## Symptom

Two independent processes (or a parent process that fans out into several
subprocesses) each mutate the same remote resource — for example, adding
and removing members of a shared group — at roughly the same time. The
remote API returns a conflict response (HTTP 409 or equivalent) for one of
the requests. The calling code classifies this as a permanent failure and
aborts the whole unit of work, even though the underlying change was
logically safe (e.g. each process was only touching its own principal).

## Cause

The mutation is logically per-item and therefore looks safe to run in
parallel, but the remote API does not offer per-item atomic updates on the
shared object — it round-trips read-modify-write on the whole resource (or
otherwise serializes writes internally) and rejects a write that raced
another one. A process-local mutex cannot see across process boundaries,
so nothing in the code actually serializes the two writers.

## Rule

1. Add an advisory file lock (e.g. `flock` on a well-known path shared by
   every process that can touch the resource) around the mutating window,
   so only one process holds the remote resource open for writes at a
   time.
2. Until the lock is proven in place, run the mutating workload strictly
   sequentially (one process at a time against the shared resource) rather
   than trusting parallelism.
3. Treat the remote API's conflict response as a transient condition, not
   a permanent failure — retry with backoff before giving up, and only
   classify as permanent after retries are exhausted.

## How to Test for It

- Reproduce without the fix: launch N processes that concurrently mutate
  the same remote resource and confirm conflict responses appear under
  load.
- Add the file lock, rerun the same N-process load, and confirm zero
  conflict responses.
- Add a unit/integration test that injects a synthetic conflict response
  into the retry path and asserts the operation succeeds after backoff
  rather than surfacing as a permanent failure.
- Audit failure classification code for this API: a conflict/409 response
  should route through the transient-retry path, never straight to
  permanent-failure handling.
