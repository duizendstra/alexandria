# go/platform/coordination/filelock

The filesystem adapter for the [coordination](../README.md) contracts: a
`Store` is a `coordination.Waiter` over one directory, so processes sharing a
filesystem queue for a window instead of mutating the same resource at once.
Standard library only, no build tags, no signal handling — the same code runs
on Linux, macOS and Windows.

## Install

The package ships inside the `go/platform/coordination` module:

```bash
go get github.com/duizendstra/alexandria/go/platform/coordination
```

## Shape

A `Store` is a directory (`Dir`), the purpose every record it writes will
state (`Purpose`), and `Options` — the jittered poll bounds and the opt-in
reclaim age (`StaleAfter`, off by default). `Acquire` is the `Waiter` method;
`Holder` returns the record an operator would read with `cat`; `LockPath`
and `FencePath` name the files a subject owns. [doc.go](doc.go) is the
operator's guide — limits, failure model, and how to operate a store —
and [example_test.go](example_test.go) is the runnable shape end to end.

## Contract

Everything below is a promise a consumer may rely on. It is the same text as
the contract section of [doc.go](doc.go);
`TestContractIsTheSameTextInREADMEAndDoc` keeps the two from drifting.

- **Exactly one caller is inside a subject's window.** `Acquire(ctx,
  subject)` blocks until it creates the subject's holder record, or `ctx` is
  done, whichever comes first. The record is staged in a file beside it,
  fsynced there, and only then published under its real name with a hard link
  — a primitive that refuses an existing target, so creating the record is
  atomic and exclusive at once.
- **Release is idempotent.** It is safe to call more than once and to defer
  unconditionally, including after an explicit call.
- **Release removes only its own record.** What a release removes is checked
  by file identity and by the record's own bytes, never by path alone — a
  path can name a successor's record, and on a filesystem that reuses inode
  numbers so can the identity: a holder that was reclaimed while still
  inside releases without touching the record of whoever holds the window
  now, so an overlap ends with the slow holder's work instead of cascading
  to a third caller.
- **The fence rises per subject.** It is read and rewritten while the window
  is held: 1 for a subject nobody has ever entered, one higher per occupancy,
  never repeated. Counters are per subject and independent of each other. The
  rewrite is fsynced — the counter file before its rename, the directory best
  effort after — before `Acquire` returns, so the fence cannot revert to an
  already-handed-out value on an unclean host crash.
- **A failed acquisition is inside nothing.** Nil release, zero fence, an
  error. The window is left as it was found; a fence that could not be
  advanced — disk full, a permission change, an unclean crash mid-rewrite —
  fails the acquisition rather than handing out an unfenced window: an
  occupancy nothing could later distinguish from any other is worse than
  making the caller retry.
- **The holder record is readable JSON.** One object: process id, host, the
  UTC instant the window was entered, and the store's stated purpose.
  `Store.Holder` returns the same record an operator reads with `cat`.
- **Three file names per subject, and the difference matters.**
  `<dir>/<subject>.lock` is the holder record and exists exactly while the
  window is held. `<dir>/<subject>.fence` is the counter and outlives every
  occupancy; deleting one silently rewinds the counter and breaks the only
  promise it makes. `<dir>/<subject>.lock.reclaimed.*` exists for
  microseconds inside any reclaim and persists only as the evidence of a
  reclaim that raced a live claim and could not restore it. An
  empty-of-`.lock` directory means nobody is inside any window.
- **Reclaim is opt-in.** `Options.StaleAfter` unset (the zero value) means a
  record is never removed on another caller's behalf, whatever its age: safe,
  and an abandoned record stalls later callers until their context ends or a
  human clears it. Set, it means a record older than that age is removed and
  the window entered: live, and a merely very slow holder is assumed dead, so
  two holders can be inside one window — which is what the fence lets a
  downstream reader notice. There is deliberately no built-in non-zero
  default.
- **Reclaim takes aside only the record it judged.** A stale record is not
  removed in place: it is set aside by rename onto a name only that reclaim
  attempt uses, and identity and bytes are checked on what actually moved. The judged
  record: reclaimed. A live claim that replaced it between the read and the
  rename: put straight back, by a primitive that refuses an existing target,
  and the wait continues behind it. If yet another claim lands inside the two
  directory operations of that gap, the restore is refused: the moved record
  is preserved under `.lock.reclaimed.*` and the wait ends with an error
  reporting that two callers are inside — the one raced outcome a filesystem
  without a conditional remove cannot rule out.
- **An unclearable abandoned record is reported, not slept through.** It ends
  the wait with an error wrapping `coordination.ErrStaleLock`.
- **A wait that gives up says whom it waited behind.** An `Acquire` that ends
  on its context wraps `ctx.Err()` and reports how long it queued and the
  holder it was queued behind — or that the window was free at the last look,
  or that the record there is unreadable.
- **The holder record is never partially visible.** Its name only ever
  appears attached to content that was already complete on disk, so a caller
  killed part-way through an acquisition leaves either no record at all or a
  whole one — never an empty file that reports no age and could therefore
  never be reclaimed.
- **An unreadable record is still a claim.** A record that exists but does
  not parse is reported as held, with age zero, so it is never reclaimed by
  accident. Since this package cannot produce such a record, one that appears
  came from outside it — damage, or a hand that did not go through this
  package — which is exactly the case where reclaiming what cannot be
  identified is the more dangerous reading.
- **Polling is jittered.** `Options.PollMin`/`PollMax` bound the wait between
  attempts; unset or inverted values fall back to the package defaults, so
  two waiters do not retry in lockstep.
- **The filesystem must support hard links.** Publishing the record with
  `os.Link` narrows the filesystems this package works on relative to
  `O_CREATE|O_EXCL`: it is fine on ext4, APFS and NTFS, and it is not
  available on FAT32 or on some network shares. A store on one of those fails
  its first acquisition rather than coordinating badly.
- **The record is stamped on the attempt that wins.** A caller that queues
  for a long time enters with a record whose age starts at the moment it
  entered, so a contended window is never reclaimable the instant it is
  acquired.
- **Advisory, single filesystem namespace.** Correctness rests on a hard link
  being atomic, and refusing an existing target, on the underlying
  filesystem. Nothing stops a process that does not go through this package,
  and nothing here can detect that it happened.

### Why it is built this way

Client-free design decisions, for the reviewer who asks "why not the obvious
alternative":

- **Link, not exclusive create.** Publishing the record with a hard link
  gives atomic-create and never-partially-visible in one primitive. An
  exclusive create needs a write after the name exists, and a kill between
  the two leaves an empty record that reports no age and can never be
  reclaimed.
- **Per-attempt staging.** The record is composed on every attempt, not once
  per wait, because it carries the instant the window was entered and
  reclaim reads that instant. Reusing one staged record across a long queue
  would stamp it at the first attempt, and a caller that queued longer than
  the reclaim age would enter already-reclaimable.
- **Identity, wherever a record is removed.** Release and reclaim both check
  file identity before removing anything, because a path alone can name a
  successor's record: an unconditional remove-by-path is exactly how a
  reclaim overlap cascades to a third caller. Identity is the dev/inode
  pair AND the record's bytes: ext4 hands a freed inode number to the very
  next file created, so a successor that claims right after a reclaim can
  sit on its predecessor's identity, and the pair alone would remove it.
  APFS never reuses an inode, which is why identity alone passes on macOS
  and fails on Linux.
- **Set-aside, not remove, in reclaim.** POSIX has no conditional remove, so
  the judged record is renamed onto a private name and identity is checked
  on what moved; a record that turns out to be live is restored by a
  primitive that refuses to clobber. What remains unavoidable is a
  microsecond, three-party window, and its outcome is preserved as evidence
  rather than silent.
- **No kernel lock.** flock would give a true conditional remove and claims
  that die with their process — and it would cost the standard-library-only,
  portable posture: Windows has no flock, so every side grows build tags and
  a second code path. The residual it would close is the microsecond window
  above. If that trade ever reverses, it belongs in a second adapter beside
  this one, not in this one growing syscalls.
- **No default reclaim age.** The safety-versus-liveness trade is priced
  where the store is bound, by the caller who knows the longest legitimate
  window; a library default would be reviewed by nobody.
- **The counter is a separate file.** The holder record must be able to
  appear and disappear atomically while the counter must survive every
  occupancy; one file cannot do both, and the two primitives differ —
  linking refuses an existing target, renaming replaces it.

## Tests

| Test | Promise |
| --- | --- |
| `TestAcquireExcludesASecondCaller` | A second acquisition for the same subject blocks until the first releases, then succeeds; the holder record is gone afterwards |
| `TestAcquireIsIdempotentOnRelease` | Calling release twice, and deferring it unconditionally, is safe |
| `TestAcquireRespectsContextCancellation` | A waiter gives up when its context ends, with an error wrapping `context.DeadlineExceeded` and a zero fence |
| `TestAcquireIsMutuallyExclusiveUnderConcurrency` | Many goroutines at one subject never show more than one holder at a time |
| `TestAcquireReclaimsAnAbandonedLock` | With reclaim switched on, a record older than the configured age does not block later callers forever, the reclaimed window still gets a fence, and no set-aside file survives the reclaim |
| `TestReleaseRemovesOnlyItsOwnRecord` | An overlapped holder's release leaves its successor's record untouched — what release removes is checked by identity, not by path |
| `TestAcquireGivingUpNamesTheHolderItWaitedBehind` | The give-up error wraps the context error and reports how long the caller queued and whom it was queued behind |
| `TestTakeAsideReclaimsTheRecordItJudged` | Reclaim takes exactly the record it judged, frees the path, and leaves no set-aside file behind |
| `TestRecordIdentityIsTheInodeAndTheBytes` | A record rewritten in place — same inode, different bytes, the inode-reuse case made deterministic — is not recognised as the judged one |
| `TestTakeAsidePutsBackALiveRecordItDidNotJudge` | A live record that replaced the judged one between the read and the set-aside is restored untouched, and the reclaim reports nothing to do |
| `TestContractIsTheSameTextInREADMEAndDoc` | The Contract section above and doc.go's contract section are the same text, mechanically |
| `TestCoordination_MutualExclusionAcrossProcesses` | **Separate processes** — the test binary re-executed as four children — queue for one subject: a counter each of them read-pauses-writes while inside ends at exactly four, and every child got a distinct fence |
| `TestCoordination_KilledHolderIsReclaimedOnlyWhenEnabled` | Against a real holder killed with SIGKILL that never released: with reclaim off a later caller waits until its own context ends; with reclaim on it enters once the record exceeds the configured age, and the fence keeps rising across the reclaim |
| `TestCoordination_FenceIsStrictlyIncreasing` | Per subject, never repeated, always higher than the occupancy before it, independent per subject, and the counter file survives release |
| `TestCoordination_HolderRecordIsReadableJSON` | The record on disk parses as the documented four-field object, names this process and the stated purpose, matches what `Store.Holder` returns, and is gone after release |
| `TestHolderRecordIsNeverPartiallyVisible` | The record is complete and parseable the moment its path exists, and the file it was staged in does not outlive the acquisition |
| `TestConcurrentAcquireStillAdmitsExactlyOne` | Publishing the record with a primitive that refuses an existing target keeps the exclusion: many goroutines at one subject, never more than one inside |
| `TestExternallyTruncatedRecordIsStillNeverReclaimed` | A record written from outside the package that cannot be read is never reclaimed on age, however old — the fail-safe survives the change |
| `TestContendedAcquireLeavesNoTempFiles` | Dozens of refused attempts leave the store holding exactly the files it owns, so a long wait is not a slow leak |
| `TestContendedAcquireIsNotBornStale` | A caller that queues for many multiples of the reclaim age enters with a fresh record, and the caller behind it finds a live claim rather than one it may take away |

The two cross-process tests re-execute the test binary (`os.Args[0]`) with an
environment marker naming the role the child should play; goroutines inside
one process cannot prove a promise about separate processes, one of which is
killed outright without ever running a deferred release.

## Out of scope / non-goals

- **Distributed locking.** Two hosts with independent local disks are not
  coordinated by this package at all, whatever path they were handed.
- **Network filesystems.** A hard link is not reliably atomic, nor reliably
  refused on an existing target, on every one of them, and nothing here can
  detect a filesystem that does not provide either property.
- **Enforcement.** Advisory only.
- **Liveness detection.** No process is ever pinged or verified; age is the
  only signal, which is why reclaim is a trade and is opt-in.
- **Refusing instead of waiting.** A `Store` is a `Waiter`. A caller that
  wants the second attempt sent home wants a `coordination.Excluder`.
- **Managing the directory.** The store creates it and its own files per
  subject; nothing else may write there, and nothing here cleans it up.
