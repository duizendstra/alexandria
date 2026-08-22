// Package filelock is the filesystem adapter for the coordination
// contracts: a [Store] is a [github.com/duizendstra/alexandria/go/platform/coordination.Waiter]
// over one directory, so processes that share a filesystem can queue for a
// window instead of mutating the same resource at once.
//
// # The contract
//
//   - Exactly one caller is inside a subject's window. Acquire(ctx,
//     subject) blocks until it creates the subject's holder record, or ctx
//     is done, whichever comes first. The record is staged in a file beside
//     it, fsynced there, and only then published under its real name with a
//     hard link — a primitive that refuses an existing target, so creating
//     the record is atomic and exclusive at once.
//   - Release is idempotent. It is safe to call more than once and to defer
//     unconditionally, including after an explicit call.
//   - Release removes only its own record. What a release removes is
//     checked by file identity, never by content or by path alone: a holder
//     that was reclaimed while still inside releases without touching the
//     record of whoever holds the window now, so an overlap ends with the
//     slow holder's work instead of cascading to a third caller.
//   - The fence rises per subject. It is read and rewritten while the
//     window is held: 1 for a subject nobody has ever entered, one higher
//     per occupancy, never repeated. Counters are per subject and
//     independent of each other. The rewrite is fsynced — the counter file
//     before its rename, the directory best effort after — before Acquire
//     returns, so the fence cannot revert to an already-handed-out value on
//     an unclean host crash.
//   - A failed acquisition is inside nothing. Nil release, zero fence, an
//     error. The window is left as it was found; a fence that could not be
//     advanced — disk full, a permission change, an unclean crash
//     mid-rewrite — fails the acquisition rather than handing out an
//     unfenced window: an occupancy nothing could later distinguish from
//     any other is worse than making the caller retry.
//   - The holder record is readable JSON. One object: process id, host, the
//     UTC instant the window was entered, and the store's stated purpose.
//     Store.Holder returns the same record an operator reads with cat.
//   - Three file names per subject, and the difference matters.
//     <dir>/<subject>.lock is the holder record and exists exactly while
//     the window is held. <dir>/<subject>.fence is the counter and outlives
//     every occupancy; deleting one silently rewinds the counter and breaks
//     the only promise it makes. <dir>/<subject>.lock.reclaimed.* exists
//     for microseconds inside any reclaim and persists only as the evidence
//     of a reclaim that raced a live claim and could not restore it. An
//     empty-of-.lock directory means nobody is inside any window.
//   - Reclaim is opt-in. Options.StaleAfter unset (the zero value) means a
//     record is never removed on another caller's behalf, whatever its age:
//     safe, and an abandoned record stalls later callers until their
//     context ends or a human clears it. Set, it means a record older than
//     that age is removed and the window entered: live, and a merely very
//     slow holder is assumed dead, so two holders can be inside one window
//     — which is what the fence lets a downstream reader notice. There is
//     deliberately no built-in non-zero default.
//   - Reclaim takes aside only the record it judged. A stale record is not
//     removed in place: it is set aside by rename onto a name only that
//     reclaim attempt uses, and identity is checked on what actually moved.
//     The judged record: reclaimed. A live claim that replaced it between
//     the read and the rename: put straight back, by a primitive that
//     refuses an existing target, and the wait continues behind it. If yet
//     another claim lands inside the two directory operations of that gap,
//     the restore is refused: the moved record is preserved under
//     .lock.reclaimed.* and the wait ends with an error reporting that two
//     callers are inside — the one raced outcome a filesystem without a
//     conditional remove cannot rule out.
//   - An unclearable abandoned record is reported, not slept through. It
//     ends the wait with an error wrapping coordination.ErrStaleLock.
//   - A wait that gives up says whom it waited behind. An Acquire that ends
//     on its context wraps ctx.Err() and reports how long it queued and the
//     holder it was queued behind — or that the window was free at the last
//     look, or that the record there is unreadable.
//   - The holder record is never partially visible. Its name only ever
//     appears attached to content that was already complete on disk, so a
//     caller killed part-way through an acquisition leaves either no record
//     at all or a whole one — never an empty file that reports no age and
//     could therefore never be reclaimed.
//   - An unreadable record is still a claim. A record that exists but does
//     not parse is reported as held, with age zero, so it is never
//     reclaimed by accident. Since this package cannot produce such a
//     record, one that appears came from outside it — damage, or a hand
//     that did not go through this package — which is exactly the case
//     where reclaiming what cannot be identified is the more dangerous
//     reading.
//   - Polling is jittered. Options.PollMin/PollMax bound the wait between
//     attempts; unset or inverted values fall back to the package defaults,
//     so two waiters do not retry in lockstep.
//   - The filesystem must support hard links. Publishing the record with
//     os.Link narrows the filesystems this package works on relative to
//     O_CREATE|O_EXCL: it is fine on ext4, APFS and NTFS, and it is not
//     available on FAT32 or on some network shares. A store on one of those
//     fails its first acquisition rather than coordinating badly.
//   - The record is stamped on the attempt that wins. A caller that queues
//     for a long time enters with a record whose age starts at the moment
//     it entered, so a contended window is never reclaimable the instant it
//     is acquired.
//   - Advisory, single filesystem namespace. Correctness rests on a hard
//     link being atomic, and refusing an existing target, on the underlying
//     filesystem. Nothing stops a process that does not go through this
//     package, and nothing here can detect that it happened.
//
// # Limits
//
// Advisory, single filesystem namespace, no distributed-locking claim.
// Nothing stops a process that does not go through this package from
// mutating the resource, and this package cannot detect that it happened.
// Correctness rests on a hard link being atomic and refusing an existing
// target on the underlying filesystem, which is true for a local disk and is
// NOT reliably true for every network filesystem. Hard links must be
// available at all: that is a narrower requirement than the exclusive create
// this package used to rely on — fine on ext4, APFS and NTFS, unavailable on
// FAT32 and on some network shares, where the first acquisition fails rather
// than coordinating badly. Two hosts with independent local disks are not
// coordinated by this package at all, whatever path they were handed.
//
// # Failure model
//
// A holder that exits without releasing — a hard kill, a crash, a machine
// that lost power — leaves its record behind. Whatever it leaves behind is a
// WHOLE record: the record is published in one step, so a process killed
// during an acquisition leaves either no record at all or a complete one,
// never an empty file that reports no age and could therefore never be
// reclaimed. What happens to the record it does leave is the caller's
// decision, not this package's:
//
//   - Options.StaleAfter unset (the zero value): reclaim is off. The record
//     is never removed on another caller's behalf and later callers wait
//     until their context ends or a human clears it. Safe, and it can stall.
//   - Options.StaleAfter set: a record older than that age is removed and
//     the window entered. Live, and it assumes a very slow holder is a dead
//     one — two holders can then be inside one window, which is exactly
//     what the fence lets a downstream reader notice. The reclaimed
//     holder's release then leaves the new record alone — release removes
//     only what its own occupancy published — so the overlap ends with the
//     slow holder's work instead of cascading to a third caller.
//
// There is deliberately no built-in non-zero default. A caller that wants
// reclaim names the age where it is bound, so that it can be reviewed
// against the longest window that is legitimately possible.
//
// An abandoned record that cannot be removed ends the wait with an error
// wrapping coordination.ErrStaleLock: a wait that could never end is
// reported, not slept through.
//
// A separate failure surface sits earlier, inside Acquire itself: the
// holder record can be created and the fence still fail to advance — disk
// full, a permission change, an unclean crash mid-rewrite. That caller is
// inside nothing: the record is removed and the error returned, never a
// window handed out without the fence that lets a downstream reader order
// occupancies. Fail-closed here is deliberate — an unfenced occupancy could
// not be told apart from any other later, which is worse than the caller
// having to retry.
//
// # Operating a store
//
// The directory is the caller's; nothing else may write there. To see who
// holds a window, read the .lock file — or call [Store.Holder], which
// returns the same record. To clear a window whose holder is provably gone,
// remove that one .lock file and leave the .fence file alone.
//
// A <subject>.lock.reclaimed.* file that persists is the evidence of a
// reclaim that raced a live claim and could not put it back: read it to see
// who was overlapped, check the fence-stamped work of that window, and
// remove the file once noted.
package filelock
