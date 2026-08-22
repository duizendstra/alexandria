# go/platform/workflow

`workflow` provides a standard-library-first, context-aware sequential step execution engine designed for multi-step CLI procedures, rollout chains, migrations, and infrastructure provisioning pipelines.

## Features

- **Sequential Step Execution**: Execute an ordered chain of named steps fail-fast upon the first non-nil error.
- **Context Awareness**: Cancellation is checked before each step; cancellation halts progression cleanly.
- **Panic Safety**: Panics within step execution are safely intercepted and returned as wrapped `ErrPanicRecovered` errors.
- **Selective Step Skipping**: Dynamic step skipping evaluated via per-step `Skip` predicates.
- **Synchronous Lifecycle Hooks**: Pluggable hooks for `OnWorkflowStart`, `OnStepStart`, `OnStepDone`, `OnStepFail`, `OnStepSkip`, and `OnWorkflowDone`.
- **Dry-Run Inspection**: Non-destructive workflow simulation via `DryRun(ctx)`.
- **Zero External Dependencies**: Implemented purely using the Go standard library.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/workflow
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duizendstra/alexandria/go/platform/workflow"
)

func main() {
	ctx := context.Background()

	wf := workflow.New("Database Migration Pipeline").
		Add("Verify Database Connectivity", func(ctx context.Context) error {
			fmt.Println("Connected to database.")
			return nil
		}).
		Add("Apply Schema Migrations", func(ctx context.Context) error {
			fmt.Println("Schema migrations applied.")
			return nil
		})

	if err := wf.Run(ctx); err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}
}
```

## SRE & Performance Hardening details

- **Sequential Execution**: Steps execute strictly sequentially on the caller's goroutine with no internal background goroutines.
- **No Inherent Retries or Resumability**: If a step returns an error, the workflow halts fail-fast. For retry policies or stateful checkpoint recovery, compose individual steps with `go/retry` and `go/platform/runstate`.
- **Context Honouring**: Context cancellation is checked before each step; a running step is not interrupted asynchronously — individual step implementations must honour the `ctx` passed to them.
- **Synchronous Lifecycle Callbacks**: Hook callbacks passed via `workflow.WithHooks` execute synchronously on the caller's goroutine. Keep hook implementations lightweight to avoid introducing latency between steps.
- **Silent Execution by Default**: `Run` produces no console output by default. For progress banners, spinner updates, or logging, attach synchronous lifecycle callbacks via `workflow.WithHooks`.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **CLI Provisioners & Migrators**: Multi-step terminal commands executing ordered setup, migration, and verification steps.
- **Background Pipeline Workers**: Sequential automation routines requiring deterministic aborts on failure.

### Load-Bearing Promises
1. **Panic Safety**: Panics originating in any step function are intercepted, wrapped into `ErrPanicRecovered`, and returned as an error rather than crashing the host process.
2. **Deterministic Short-Circuiting**: Execution stops immediately at the first non-nil error; subsequent steps are never invoked.
3. **Context Propagation**: Cancellation or timeout of the passed `context.Context` halts progression before each subsequent step.
4. **Skip Semantics**: A step whose `Skip` predicate returns true is recorded as `StatusSkipped`, `OnStepSkip` fires, and neither `OnStepStart` nor `OnStepFail` is invoked.
