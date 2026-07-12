package async

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	// StatusPending means the task is queued but not yet started.
	StatusPending Status = "pending"
	// StatusRunning means the task is currently executing.
	StatusRunning Status = "running"
	// StatusDone means the task completed successfully.
	StatusDone Status = "done"
	// StatusFailed means the task completed with an error.
	StatusFailed Status = "failed"
)

// Result holds the typed outcome of a completed task.
type Result struct {
	// Data is the operation-specific result (e.g. quality gate output).
	Data any
	// Error is the human-readable error message when Status is Failed.
	Error string
}

// Task tracks a single unit of async work.
type Task struct {
	ID        string
	Kind      string // Operation type (e.g. "quality", "build", "deploy").
	Status    Status
	Created   time.Time
	Started   time.Time
	Completed time.Time
	Result    Result
}

// Runner manages async task execution with in-memory state.
type Runner struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	sem   chan struct{}
}

const defaultConcurrencyLimit = 100

// NewRunner creates a task runner with a default concurrency limit of 100.
func NewRunner() *Runner {
	return NewRunnerWithLimit(defaultConcurrencyLimit)
}

// NewRunnerWithLimit creates a task runner with a custom concurrency limit.
// A limit <= 0 means unbounded concurrency (no worker pool limit).
func NewRunnerWithLimit(limit int) *Runner {
	var sem chan struct{}
	if limit > 0 {
		sem = make(chan struct{}, limit)
	}

	return &Runner{
		tasks: make(map[string]*Task),
		sem:   sem,
	}
}

// Submit queues a function for async execution and returns the task ID immediately.
// The kind parameter identifies the operation type for filtering (e.g. "quality").
func (r *Runner) Submit(kind string, fn func() (any, error)) string {
	id := newID()
	t := &Task{
		ID:      id,
		Kind:    kind,
		Status:  StatusPending,
		Created: time.Now(),
	}

	r.mu.Lock()
	r.tasks[id] = t
	r.mu.Unlock()

	go r.execute(t, fn)

	return id
}

// Get returns a snapshot of the task with the given ID.
// Returns nil if the task does not exist.
func (r *Runner) Get(id string) *Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tasks[id]
	if !ok {
		return nil
	}

	// Return a copy so callers can't mutate internal state.
	cp := *t

	return &cp
}

// List returns value-copied snapshots of all tasks to prevent heap allocation of pointers.
func (r *Runner) List() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		result = append(result, *t)
	}

	return result
}

// Prune removes completed tasks older than the given age.
// Returns the number of tasks removed.
func (r *Runner) Prune(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, t := range r.tasks {
		if t.Status != StatusPending && t.Status != StatusRunning && t.Completed.Before(cutoff) {
			delete(r.tasks, id)
			removed++
		}
	}

	return removed
}

func (r *Runner) execute(t *Task, fn func() (any, error)) {
	if r.sem != nil {
		r.sem <- struct{}{}
		defer func() { <-r.sem }()
	}

	r.mu.Lock()
	t.Status = StatusRunning
	t.Started = time.Now()
	r.mu.Unlock()

	var data any
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			r.mu.Lock()
			t.Completed = time.Now()
			t.Status = StatusFailed
			t.Result = Result{
				Error: fmt.Sprintf("panic: %v", rec),
			}
			r.mu.Unlock()
		}
	}()

	data, err = fn()

	r.mu.Lock()
	t.Completed = time.Now()

	if err != nil {
		t.Status = StatusFailed
		t.Result = Result{Data: data, Error: err.Error()}
	} else {
		t.Status = StatusDone
		t.Result = Result{Data: data}
	}

	r.mu.Unlock()
}

const idBytes = 8 // 16 hex characters.

func newID() string {
	var b [idBytes]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(b[:])
}
