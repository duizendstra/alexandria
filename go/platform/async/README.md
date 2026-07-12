# go/platform/async

`go/platform/async` provides an SRE-hardened, panic-resilient in-memory asynchronous task runner with bounded concurrency and task status lifecycle tracking.

## Features

- **Bounded Concurrency**: Controls goroutine spawn rates using customizable semaphore pools.
- **Panic Protection**: Built-in deferred recovery catches task panics, recording failure details without crashing the host process.
- **Lifecycle Tracking**: Tracks task states from `pending` to `running`, `done`, or `failed`.
- **Thread-Safe Snapshots**: Returns copied task objects to prevent shared state mutation or race conditions.
- **Age-Based Pruning**: Prunes old execution records from memory to prevent memory leaks over time.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/async
```

## Quick Start

### Executing Tasks Asynchronously with Concurrency Control

```go
package main

import (
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/platform/async"
)

func main() {
	// Initialize a runner with a maximum of 10 concurrent worker goroutines
	runner := async.NewRunnerWithLimit(10)

	// Submit a task for execution
	taskID := runner.Submit("backup", func() (any, error) {
		// Simulate background task
		time.Sleep(50 * time.Millisecond)
		return "Backup completed successfully", nil
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

	// Prune completed tasks older than 1 minute to release memory
	prunedCount := runner.Prune(1 * time.Minute)
	fmt.Printf("Pruned %d task(s)\n", prunedCount)
}
```

## SRE & Performance Hardening details

1. **Host Crash Prevention**: Background goroutine failures are caught using recovery blocks. If a task panics, the runner safely logs the panic output to the task result and frees the concurrency semaphore rather than crashing the entire microservice.
2. **Goroutine Leak Protection**: Utilizes a buffered channel semaphore to bound concurrency. When the concurrency limit is reached, pending tasks queue up rather than spawning unlimited, resource-heavy goroutines.
3. **Task State Copying**: Querying task states via `Get` and `List` performs field copying of `Task` structs. This preserves internal runner thread-safety and avoids read/write races on the internal state map.
