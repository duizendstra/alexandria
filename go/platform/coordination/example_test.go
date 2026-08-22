package coordination_test

import (
	"errors"
	"fmt"
	"sync"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

// ExampleSubject_Validate shows the one rule a subject must satisfy: it
// becomes part of a path, so it may not escape the store it is stored in.
// Validate is called by every adapter, but a caller that composes subjects
// from data can check earlier and report a better error.
func ExampleSubject_Validate() {
	for _, s := range []coordination.Subject{"shared-index", "region/eu", "..", ""} {
		err := s.Validate()
		switch {
		case err == nil:
			fmt.Printf("%-14q ok\n", s)
		case errors.Is(err, coordination.ErrBadSubject):
			fmt.Printf("%-14q rejected\n", s)
		}
	}

	// Output:
	// "shared-index" ok
	// "region/eu"    rejected
	// ".."           rejected
	// ""             rejected
}

// memoryExcluder is a minimal in-process [coordination.Excluder]: enough to
// show the semantics, and deliberately not a package anybody should use —
// a real one refuses across processes, not just across goroutines.
type memoryExcluder struct {
	mu   sync.Mutex
	held map[coordination.Subject]bool
}

func (e *memoryExcluder) TryAcquire(subject coordination.Subject) (func(), error) {
	if err := subject.Validate(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.held[subject] {
		return nil, fmt.Errorf("%s: %w", subject, coordination.ErrLocked)
	}

	if e.held == nil {
		e.held = map[coordination.Subject]bool{}
	}
	e.held[subject] = true

	var once sync.Once

	return func() {
		once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.held, subject)
		})
	}, nil
}

// ExampleExcluder shows the other half of the published language, and the
// decision behind it: an Excluder sends the second caller home with
// ErrLocked instead of queueing it. That is the right answer when a second
// concurrent attempt is a mistake in itself — a duplicate run, a double
// submission — and the operator is better served by hearing about it now.
// Classify the refusal with errors.Is; never by matching the text.
func ExampleExcluder() {
	var excluder coordination.Excluder = &memoryExcluder{}

	release, err := excluder.TryAcquire("nightly-rebuild")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	_, err = excluder.TryAcquire("nightly-rebuild")
	fmt.Println("second attempt refused:", errors.Is(err, coordination.ErrLocked))

	release()

	release2, err := excluder.TryAcquire("nightly-rebuild")
	fmt.Println("after release:", err == nil)
	release2()

	// Output:
	// second attempt refused: true
	// after release: true
}
