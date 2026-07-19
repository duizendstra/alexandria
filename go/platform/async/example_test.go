package async_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/platform/async"
)

var errSinkUnavailable = errors.New("sink unavailable")

func ExampleRunner() {
	r := async.NewRunner(async.WithLimit(4))
	defer func() { _ = r.Close() }()

	// Submit returns immediately with a task ID; the function runs in the
	// background and should honour ctx cancellation.
	id := r.Submit("greet", func(_ context.Context) (any, error) {
		return "hello", nil
	})

	// Poll until the task completes.
	for {
		t := r.Get(id)
		if t.Status == async.StatusDone || t.Status == async.StatusFailed {
			fmt.Println(t.Status, t.Result.Data)

			break
		}

		time.Sleep(time.Millisecond)
	}
	// Output: done hello
}

func ExampleNewBatchBuffer() {
	ctx := context.Background()

	// Flush automatically once two items are buffered.
	buf := async.NewBatchBuffer(2, func(_ context.Context, batch []string) error {
		fmt.Println("flushed:", batch)

		return nil
	})

	_ = buf.Add(ctx, "a")
	_ = buf.Add(ctx, "b") // Limit reached: triggers an automatic flush.
	_ = buf.Add(ctx, "c")

	// Flush delivers whatever is still buffered.
	_ = buf.Flush(ctx)
	// Output:
	// flushed: [a b]
	// flushed: [c]
}

func ExampleFlushError() {
	ctx := context.Background()

	buf := async.NewBatchBuffer(10, func(_ context.Context, _ []int) error {
		return errSinkUnavailable
	})

	_ = buf.Add(ctx, 1)
	_ = buf.Add(ctx, 2)

	// The buffer does not retain a failed batch; the FlushError carries it
	// so the caller can retry or salvage the items.
	err := buf.Flush(ctx)

	var flushErr *async.FlushError[int]
	if errors.As(err, &flushErr) {
		fmt.Println(flushErr.Items, errors.Is(err, errSinkUnavailable))
	}
	// Output: [1 2] true
}
