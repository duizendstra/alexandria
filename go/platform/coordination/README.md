# go/platform/coordination

Interfaces and primitives for process and workload mutual exclusion and coordination.

## Excluder

`Excluder` defines the canonical contract for mutual exclusion per named subject:

```go
type Excluder interface {
    Acquire(subject string) (func(), error)
}
```

When an acquisition is blocked by another concurrent process holding the lock, implementations return `coordination.ErrLocked`.

## Implementations

- **`filelock`** (`go/platform/coordination/filelock`): Lockfile-based mutual exclusion with atomic `O_CREATE|O_EXCL` and signal cleanup.
- **`NopExcluder()`**: In-memory no-op excluder for testing or single-run configurations.

## Install

```bash
go get github.com/duizendstra/alexandria/go/platform/coordination
```
