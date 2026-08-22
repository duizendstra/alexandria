package filelock_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/duizendstra/alexandria/go/platform/coordination"
	"github.com/duizendstra/alexandria/go/platform/coordination/filelock"
)

func ExampleLocker() {
	dir, err := os.MkdirTemp("", "filelock-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	locker := &filelock.Locker{Dir: dir}

	release, err := locker.Acquire("job-alpha")
	if err != nil {
		if errors.Is(err, coordination.ErrLocked) {
			fmt.Println("job is already running")
			return
		}
		panic(err)
	}
	defer release()

	fmt.Println("lock acquired successfully")
	// Output:
	// lock acquired successfully
}
