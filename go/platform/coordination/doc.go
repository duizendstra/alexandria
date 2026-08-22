// Package coordination is the published language for keeping two or more
// processes out of each other's way while one of them mutates a resource
// they share.
//
// It holds contracts only — subject, holder record, the two acquisition
// interfaces and the error vocabulary — and depends on nothing but the
// standard library. Adapters live in subpackages (see
// coordination/filelock) or in the consumer's own infrastructure layer;
// nothing here touches a filesystem, a clock or a network.
//
// # The model
//
// A [Subject] names one window: the smallest stretch of work during which a
// shared resource may not be mutated by anybody else. A [Holder] record says
// who is inside that window right now, and since when. Two acquisition
// styles are offered, and the difference between them is a deliberate
// product decision, not an implementation detail:
//
//   - A [Waiter] makes the second caller queue. Use it when both callers
//     have legitimate work to do and the only problem is that they must not
//     do it at the same instant.
//   - An [Excluder] sends the second caller home with [ErrLocked]. Use it
//     when a second attempt is a mistake in itself — a duplicate run, a
//     double submission — and waiting would only delay the operator's
//     discovery of it.
//
// # The window is the mutation, not the workflow
//
// The single most common misuse of a coordination primitive is to hold it
// across a whole job because that is easier to reason about. Do not. Hold
// it around the mutation that actually conflicts and release it before the
// long, independent work that follows. A window that spans a whole workflow
// turns two concurrent operators into one, converts every crash into a stall
// for everybody, and makes the reclaim trade-off below far more dangerous
// than it needs to be.
//
// # Single host, advisory only
//
// These contracts describe advisory coordination between processes that
// share one filesystem namespace and one wall clock: several processes on
// one machine, or on a shared volume where atomic-create semantics hold.
// They make no distributed-locking claim. Nothing prevents a process that
// does not go through this package from mutating the same resource, and no
// adapter here can detect that it happened. If mutual exclusion has to hold
// across independent hosts with independent clocks, this is the wrong
// package: use a service that can fence and revoke, not a file.
//
// # The reclaim trade-off
//
// A holder that is killed hard, or whose machine loses power, leaves its
// record behind and never releases. An adapter may be configured to treat
// such a record as abandoned after some age and reclaim it. That is a
// trade, and it must be made explicitly by the composition root, never
// defaulted on by a library:
//
//   - Reclaim off (the recommended default): safety first. A window is
//     never entered while another record claims it, whatever that record's
//     age. The cost is that an abandoned record blocks every later caller
//     until a human removes it.
//   - Reclaim on: liveness first. After the configured age the record is
//     removed and the window is entered. The cost is that a holder which is
//     merely very slow — a stalled network call, a paused process — is
//     assumed dead while it is still alive, and two holders can then be
//     inside one window at once. When that happens, the reclaimed holder's
//     release leaves the new record alone — an adapter removes only what
//     its own occupancy published — so the overlap ends with the slow
//     holder's work instead of cascading to a third caller.
//
// Pick the age deliberately: it must exceed the longest legitimate window
// by a wide margin, so that "older than this" really does mean "gone", and
// it must be passed in at the binding site where somebody can review it,
// which is why no adapter in this package supplies a default. When an
// adapter finds an abandoned record it cannot clear, it reports
// [ErrStaleLock] rather than looping in silence.
//
// # The fence
//
// [Waiter.Acquire] also returns a fence: a counter, strictly increasing per
// subject, advanced while the window is held. Two acquisitions of the same
// subject never share a fence value, and a later acquisition always carries
// a higher one than an earlier one.
//
// The fence exists because reclaim makes overlap possible in principle. It
// lets everything downstream — an audit record, an evidence file, a
// receiving system — tell one occupancy of a window from the next, and lets
// a reader discard work stamped with a fence lower than one it has already
// seen. Record it wherever the work inside the window is recorded. It is a
// number for correlation and ordering, not a token to authenticate with,
// and it is not a substitute for holding the window.
//
// # Errors
//
// [ErrBadSubject] rejects a name that cannot safely become part of a path.
// [ErrLocked] is an [Excluder]'s refusal. [ErrStaleLock] reports a record
// left behind by a holder that is gone. All three are sentinels: classify
// with errors.Is, never by matching text.
package coordination
