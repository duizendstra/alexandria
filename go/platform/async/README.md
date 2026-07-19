# go/platform/async

`go/platform/async` provides an SRE-hardened, panic-resilient in-memory asynchronous task runner with bounded concurrency and task status lifecycle tracking.

## Features

- **Bounded Concurrency with Backpressure**: The concurrency slot is acquired in `Submit`, so a saturated runner blocks the submitter instead of accumulating one parked goroutine per pending task.
- **Context Propagation**: Every task function receives a `context.Context` derived from the runner's base context; `Close()` cancels all outstanding task contexts.
- **Panic Protection**: Built-in deferred recovery catches task panics, recording failure details without crashing the host process.
- **Lifecycle Tracking**: Tracks task states from `pending` to `running`, `done`, or `failed`.
- **Thread-Safe Snapshots**: Returns copied task objects to prevent shared state mutation or race conditions.
- **Age-Based Pruning**: Prunes old execution records manually via `Prune` or automatically via the optional `WithJanitor` background reaper.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/async
```

## Quick Start

### Executing Tasks Asynchronously with Concurrency Control

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/platform/async"
)

func main() {
	// Initialize a runner with a maximum of 10 concurrent worker goroutines
	// and a janitor that reaps completed tasks older than 1 minute.
	runner := async.NewRunner(
		async.WithLimit(10),
		async.WithJanitor(30*time.Second, 1*time.Minute),
	)
	// Close cancels the context of every outstanding task and stops the janitor.
	defer runner.Close()

	// Submit a task for execution. The context is derived from the runner's
	// base context and is cancelled when the runner is closed.
	taskID := runner.Submit("backup", func(ctx context.Context) (any, error) {
		// Simulate background task, honouring cancellation.
		select {
		case <-time.After(50 * time.Millisecond):
			return "Backup completed successfully", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	fmt.Printf("Task submitted. ID: %s\n", taskID)

	// Poll status until completion
	for {
		task := runner.Get(taskID)
		if task == nil {
			break
		}

		if task.Status == async.StatusDone || task.Status == async.StatusFailed {
			fmt.Printf("Task finished. Status: %s, Data: %v, Err: %q\n",
				task.Status, task.Result.Data, task.Result.Error)
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	// The janitor prunes completed tasks in the background; Prune can still
	// be called manually to release memory immediately.
	prunedCount := runner.Prune(1 * time.Minute)
	fmt.Printf("Pruned %d task(s)\n", prunedCount)
}
```

## SRE & Performance Hardening details

1. **Host Crash Prevention**: Background goroutine failures are caught using recovery blocks. If a task panics, the runner safely logs the panic output to the task result and frees the concurrency semaphore rather than crashing the entire microservice.
2. **Goroutine Leak Protection**: Utilizes a buffered channel semaphore to bound concurrency. The slot is acquired in `Submit` itself, so when the limit is reached submitters block (backpressure) rather than spawning unlimited, resource-heavy goroutines that park on the semaphore.
3. **Task State Copying**: Querying task states via `Get` and `List` performs field copying of `Task` structs. This preserves internal runner thread-safety and avoids read/write races on the internal state map.
