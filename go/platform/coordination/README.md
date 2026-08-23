# go/platform/coordination

The published language for keeping two or more processes out of each other's
way while one of them mutates a resource they share. Contracts only — no
filesystem, no clock, no network, standard library only. Adapters live in
subpackages ([filelock](filelock/README.md)) or in a consumer's own
infrastructure layer.

## Install

```bash
go get github.com/duizendstra/alexandria/go/platform/coordination
```

## Shape

- `Subject` names one window; `Subject.Validate` keeps it safe to use as a
  path component.
- `Waiter.Acquire(ctx, subject)` queues for the window and returns a release,
  the occupancy's fence, or an error.
- `Excluder.TryAcquire(subject)` refuses a second caller with `ErrLocked`
  instead of queueing.
- `Holder` is the readable record of who is inside; `Self` fills it in for the
  calling process; `Holder.Age` says how long it has been there.
- `ErrBadSubject`, `ErrLocked` and `ErrStaleLock` are the sentinels; classify
  with `errors.Is`.

[doc.go](doc.go) is the operator's guide to choosing between the two styles
and pricing reclaim; [example_test.go](example_test.go) is the runnable shape.

## Contract

Everything below is a promise a consumer may rely on. [doc.go](doc.go)
states the same promises as connected prose; this list is the checklist
form, and a disagreement between the two is a defect.

- **A subject names one window.** `Subject` is the smallest stretch of work
  during which the shared resource may not be mutated by anybody else. It
  becomes part of a path, so `Subject.Validate` rejects the empty string,
  `.`, `..`, anything holding a path separator and anything holding a parent
  reference, with an error wrapping `ErrBadSubject`.
- **Two acquisition styles, and the difference is a product decision.**
  `Waiter.Acquire` makes the second caller queue — use it when both callers
  have legitimate work and the only problem is simultaneity. `Excluder.TryAcquire`
  refuses the second caller with `ErrLocked` — use it when a second
  concurrent attempt is a mistake in itself and the operator is better served
  by hearing about it now.
- **Release is idempotent.** Both styles return a release that is safe to
  call more than once and safe to defer unconditionally.
- **A failed acquisition is inside nothing.** Nil release, zero fence, an
  error; the window is left as it was found.
- **The window is the mutation, not the workflow.** Hold it around the change
  that actually conflicts and leave before the long independent work that
  follows. A window spanning a whole workflow turns two concurrent operators
  into one and turns every crash into a stall for everybody.
- **The holder record is readable.** `Holder` says which process on which
  host entered the window, at what UTC instant, and for what stated purpose.
  `Self` fills it in for the calling process. An operator with nothing but
  `cat` can see who is inside.
- **`Holder.Age` is zero without `Since`.** An incomplete or unreadable
  record has no meaningful age, so nothing can mistake it for an abandoned
  one.
- **Reclaim is opt-in and is a trade, not a default.** Reclaim off: a window
  is never entered while another record claims it, whatever its age — safe,
  and an abandoned record stalls later callers until a human clears it.
  Reclaim on: a record older than a configured age is removed and the window
  entered — live, and a merely very slow holder is assumed dead, so two
  holders can be inside one window. The age is stated at the composition
  root, where it can be reviewed against the longest legitimate window; no
  adapter supplies one.
- **A release never removes a successor's claim.** When reclaim does put two
  holders inside one window, the reclaimed holder's release leaves the new
  holder's record untouched — an adapter removes only what the releasing
  occupancy itself published — so the overlap ends with the slow holder's
  work instead of cascading to a third caller.
- **The fence orders occupancies.** `Waiter.Acquire` returns a per-subject
  counter, advanced while the window is held: two occupancies never share a
  value, a later one is always higher. It exists because reclaim makes
  overlap possible in principle, and it lets everything downstream tell one
  occupancy from the next and discard work stamped lower than what it has
  already seen. It is for correlation and ordering — not a token to
  authenticate with, and not a substitute for holding the window.
- **Advisory, single filesystem namespace, single clock.** No
  distributed-locking claim. Nothing stops a process that does not go through
  this package from mutating the resource, and nothing here can detect that
  it happened.
- **Errors are sentinels.** `ErrBadSubject`, `ErrLocked`, `ErrStaleLock`;
  classify with `errors.Is`, never by matching text.

## Tests

| Test | Promise |
| --- | --- |
| `TestCoordination_SubjectRejectsAnythingThatEscapesTheStore` | A name that could leave the store is refused with `ErrBadSubject`; ordinary names are accepted |
| `TestCoordination_SubjectErrorNamesTheOffendingSubject` | The refusal is diagnosable: the rejected name is in the message |
| `TestCoordination_SelfDescribesTheCallingProcess` | The record written for the calling process carries this pid, a non-empty host, the stated purpose, and a UTC instant between the call's before and after |
| `TestCoordination_HolderAgeIsZeroWithoutSince` | An incomplete record has no meaningful age, so it can never be mistaken for an abandoned one |
| `TestCoordination_HolderStringIsOneOperatorReadableLine` | The record renders as one line naming pid, host, UTC instant and purpose; an unstated purpose is said, not left blank |
| `TestCoordination_HolderRoundTripsAsJSON` | The record's wire shape: four documented fields under their documented names, surviving a round trip |
| `TestCoordination_ErrorsAreDistinctSentinels` | The three published errors can be told apart with `errors.Is` |

The adapter's own tests — including the ones that re-execute the test binary
as separate processes — are listed in
[filelock/README.md](filelock/README.md#tests).

## Out of scope / non-goals

- **Distributed locking.** Independent hosts with independent clocks and
  independent disks are not coordinated by anything here. If mutual exclusion
  must hold across hosts, use a service that can fence and revoke.
- **Multi-host or network-filesystem correctness.** Atomic-create semantics
  are not reliably atomic on every network filesystem; this package cannot
  detect a filesystem that does not provide them.
- **Enforcement.** Coordination here is advisory. A process that ignores the
  package mutates the resource unopposed, and no adapter can notice.
- **Liveness detection.** Nothing here pings a holder or verifies a process
  still exists. Age is the only signal, which is exactly why reclaim is a
  trade and is opt-in.
- **Revocation.** A holder is never told it lost its window. The fence lets a
  downstream reader detect the consequence; it does not prevent it.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Adapter authors**: infrastructure code implementing an acquisition
  interface over a file, a database row, or an object-store precondition.
- **Callers serialising mutation**: processes that must not both mutate a
  shared resource, and need a holder record an operator can read.

### Load-Bearing Promises
1. **Contracts Only, Standard Library Only**: this module holds the subject,
   the holder record, the two acquisition interfaces and the error vocabulary,
   and depends on nothing outside the standard library. Adapters live
   elsewhere; a dependency added here is a breaking change to every consumer.
2. **A Subject Cannot Escape Its Store**: a subject that would address
   something outside the store is rejected, and the error names the offending
   subject rather than failing anonymously.
3. **Errors Are Distinct Sentinels**: each failure mode is separately matchable
   with `errors.Is`, so a caller can tell "held by someone else" from "cannot
   reach the store".
4. **A Holder Round-Trips As JSON**: the record survives being written by one
   process and read by another.
5. **A Holder Prints As One Operator-Readable Line**: `String` is a single
   line, safe to put straight into a log or an error message.
6. **Age Is Zero Without `Since`**: an unset timestamp reports zero age rather
   than an epoch-derived nonsense value.
