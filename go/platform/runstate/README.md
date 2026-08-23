# go/platform/runstate

The on-disk state a run needs to be safe to repeat: an exclusive **lock** per
subject, and a short-lived **lease** that proves a check was passed.

```go
locker := &runstate.Locker{Dir: stateDir}
leases := &runstate.LeaseStore{Dir: stateDir}
```

## Lock

One run per subject at a time. The lock file is created with
`O_CREATE|O_EXCL`, so two processes cannot both think they hold it, and it is
cleaned up on release *and* on SIGINT/SIGTERM — a lock left behind by a Ctrl-C
would block the next run for no reason.

```go
release, err := locker.Acquire("job-alpha")
if errors.Is(err, runstate.ErrLocked) {
    // another run is busy with this subject
}
defer release() // safe to call more than once
```

`Locker` is a [`coordination.Excluder`](../coordination/): `TryAcquire` takes a
`coordination.Subject` and refuses a second caller with the coordination
sentinel. `runstate.ErrLocked` and `runstate.ErrBadSubject` *are*
`coordination.ErrLocked` and `coordination.ErrBadSubject`, so `errors.Is`
matches under either name and a caller never needs a translation layer.

```go
release, err := locker.TryAcquire(coordination.Subject("job-alpha"))
if errors.Is(err, coordination.ErrLocked) {
    // the same refusal, seen through the contract
}
```

On an interrupt the lock file is removed and the signal is re-raised, so the
shell sees the usual `128+n` exit code. Windows has no self-signal facility;
there the process exits with code 1 after the cleanup instead.

## Lease

A lease says: *this subject passed its check, against this fingerprint, at this
time.*

```go
// The check passed.
_ = leases.Save(runstate.Lease{
    Subject:     "job-alpha",
    Fingerprint: buildCommit,     // a commit, a content hash, a config digest
    IssuedAt:    time.Now(),
})

// Later, before doing the real work.
lease, found, err := leases.Load("job-alpha")
if !found || !lease.Valid(time.Now(), "job-alpha", buildCommit, time.Hour) {
    // refuse: run the check again first
}

_ = leases.Consume("job-alpha") // spent after a successful run
```

The **fingerprint** is what makes a lease safe to keep on disk. A lease issued
against one fingerprint says nothing about another, so a rebuild or a config
change invalidates it without anyone having to remember to clean up. Compare a
plain timestamp file, which keeps vouching for whatever the code happens to be
now.

Three deliberate choices:

- **A lease dated in the future is invalid.** Clock skew is the one case where
  a stale lease could outlive its window, and a run that refuses is cheaper
  than one that should not have started.
- **An unreadable lease is no lease, not an error.** The answer that makes the
  caller redo the check is safer than one that makes it stop.
- **`Save` is atomic** (temporary file, then rename), so a reader never sees a
  half-written lease, and `Consume` is idempotent — consuming after a success
  and revoking after a failure are the same call.

A subject becomes part of a file name, so one that contains a path separator or
a parent reference is refused with `ErrBadSubject`.

The only dependency is [`go/platform/coordination`](../coordination/), for the
`Excluder` contract and its sentinels.

## Install

```bash
go get github.com/duizendstra/alexandria/go/platform/runstate
```

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- Command-line tools with a two-step ceremony — plan, then apply — where the
  apply step must refuse to run without a recent plan against the same build.

### Load-Bearing Promises
1. **A Lease Needs A Fingerprint On Both Sides**: a lease with an empty
   fingerprint on either side is invalid. An apply cannot be matched to a plan
   by accident.
2. **Saving Is Atomic**: a lease is written atomically, so an interrupted save
   leaves the previous state rather than a truncated file.
3. **An Unreadable Lease Is Ignored, Not Fatal**: corrupt state degrades to
   "no lease" instead of breaking the tool that finds it.
4. **A Lease Is Readable JSON**: an operator can inspect state with ordinary
   tools rather than needing the program that wrote it.
5. **Subjects Cannot Escape**: a subject that would address something outside
   the store is refused; ordinary subjects are accepted.
6. **The Locker Excludes A Second Run**: a second concurrent run is kept out,
   and an unusable state directory is reported rather than silently treated as
   "no lock held".

