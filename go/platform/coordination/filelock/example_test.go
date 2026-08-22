package filelock_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
	"github.com/duizendstra/alexandria/go/platform/coordination/filelock"
)

// Example shows the intended shape end to end: the policy owner allocates
// the store and decides the reclaim trade, the executor enters one window
// with a stated purpose, reads its fence, and defers the release
// unconditionally.
func Example() {
	// --- the policy owner: allocates the store, states the trade ---------
	//
	// The directory is a decision about layout, and StaleAfter is a decision
	// about safety versus liveness. Both are made here, once, where they can
	// be reviewed — never defaulted on deeper down.
	root, err := os.MkdirTemp("", "coordination-example")
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	defer os.RemoveAll(root)

	store := &filelock.Store{
		Dir:     filepath.Join(root, "coordination"),
		Purpose: "updating the shared index",
		Options: filelock.Options{StaleAfter: 10 * time.Minute},
	}

	// --- the executor: enters one window, records the fence --------------
	//
	// The window is the mutation, not the workflow: it is entered
	// immediately before the conflicting change and left immediately after,
	// with the long independent work on either side of it.
	enter := func(subject coordination.Subject) (uint64, error) {
		release, fence, err := store.Acquire(context.Background(), subject)
		if err != nil {
			return 0, err
		}
		defer release()

		// ... the mutation nobody else may make at the same time ...

		return fence, nil
	}

	first, err := enter("shared-index")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	second, err := enter("shared-index")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Printf("fences: %d then %d\n", first, second)

	// Output:
	// fences: 1 then 2
}

// ExampleStore_Holder shows what an operator finds in the store while a
// window is held: one readable JSON record naming the process, the host,
// the instant and the purpose.
func ExampleStore_Holder() {
	root, err := os.MkdirTemp("", "coordination-holder")
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	defer os.RemoveAll(root)

	store := &filelock.Store{Dir: root, Purpose: "updating the shared index"}

	release, _, err := store.Acquire(context.Background(), "shared-index")
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	defer release()

	// Either read the record straight off the disk...
	path, _ := store.LockPath("shared-index")
	raw, _ := os.ReadFile(path)
	var onDisk map[string]any
	_ = json.Unmarshal(raw, &onDisk)

	// ...or ask the store for it.
	holder, held, err := store.Holder("shared-index")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("file name:", filepath.Base(path))
	fmt.Println("record keys:", len(onDisk))
	fmt.Println("held:", held, "purpose:", holder.Purpose)

	// Output:
	// file name: shared-index.lock
	// record keys: 4
	// held: true purpose: updating the shared index
}
