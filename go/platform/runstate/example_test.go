package runstate_test

import (
	"fmt"
	"os"
	"time"

	"github.com/duizendstra/alexandria/go/platform/runstate"
)

// exampleSubject is the subject these examples work with.
const exampleSubject = "job-alpha"

// A lease is proof that a check was passed, bound to the subject it was passed
// for and to a fingerprint of what was checked. Rebuild the thing and the
// fingerprint changes, so the lease stops applying without anyone having to
// clean it up.
func ExampleLease_Valid() {
	now := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)
	lease := runstate.Lease{
		Subject:     exampleSubject,
		Fingerprint: "1a2b3c4d",
		IssuedAt:    now.Add(-10 * time.Minute),
	}

	fmt.Println("same build: ", lease.Valid(now, exampleSubject, "1a2b3c4d", time.Hour))
	fmt.Println("after rebuild:", lease.Valid(now, exampleSubject, "99887766", time.Hour))
	fmt.Println("other subject:", lease.Valid(now, "job-beta", "1a2b3c4d", time.Hour))
	fmt.Println("too old:    ", lease.Valid(now.Add(2*time.Hour), exampleSubject, "1a2b3c4d", time.Hour))
	// Output:
	// same build:  true
	// after rebuild: false
	// other subject: false
	// too old:     false
}

// The store keeps one lease per subject. A dry run issues it, the real run
// checks it and consumes it.
func ExampleLeaseStore() {
	dir, _ := os.MkdirTemp("", "runstate")
	defer func() { _ = os.RemoveAll(dir) }()

	store := &runstate.LeaseStore{Dir: dir}
	now := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)

	// The check passed: issue the lease.
	_ = store.Save(runstate.Lease{Subject: exampleSubject, Fingerprint: "1a2b3c4d", IssuedAt: now})

	// The real run, a minute later.
	lease, found, _ := store.Load(exampleSubject)
	fmt.Println("may run:", found && lease.Valid(now.Add(time.Minute), exampleSubject, "1a2b3c4d", time.Hour))

	// It ran: the lease is spent.
	_ = store.Consume(exampleSubject)

	_, found, _ = store.Load(exampleSubject)
	fmt.Println("still there:", found)
	// Output:
	// may run: true
	// still there: false
}

// A lock keeps a second run of the same subject out for as long as the first
// one lasts.
func ExampleLocker_Acquire() {
	dir, _ := os.MkdirTemp("", "runstate")
	defer func() { _ = os.RemoveAll(dir) }()

	locker := &runstate.Locker{Dir: dir}

	release, err := locker.Acquire(exampleSubject)
	if err != nil {
		fmt.Println("could not start:", err)

		return
	}
	defer release()

	_, err = locker.Acquire(exampleSubject)
	fmt.Println("second run refused:", err != nil)
	// Output: second run refused: true
}
