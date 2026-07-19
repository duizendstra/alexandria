package async

import (
	"context"
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
//
// Every task function receives a context derived from the runner's base
// context; Close cancels it, signalling outstanding tasks to stop. Call
// Close when done with the runner to release the base context and stop the
// optional background janitor.
type Runner struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	sem    chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

const defaultConcurrencyLimit = 100

// Option configures a Runner created by NewRunner.
type Option func(*runnerOptions)

type runnerOptions struct {
	limit           int
	baseCtx         context.Context
	janitorInterval time.Duration
	janitorMaxAge   time.Duration
}

// WithLimit sets the concurrency limit (default 100).
// A limit <= 0 means unbounded concurrency (no worker pool limit).
func WithLimit(limit int) Option {
	return func(o *runnerOptions) {
		o.limit = limit
	}
}

// WithBaseContext sets the base context from which every per-task context is
// derived. Cancelling it cancels all outstanding task contexts, exactly like
// Close. Defaults to context.Background().
func WithBaseContext(ctx context.Context) Option {
	return func(o *runnerOptions) {
		o.baseCtx = ctx
	}
}

// WithJanitor starts a background worker that removes completed tasks older
// than maxAge every interval, so callers do not need to call Prune manually.
// The janitor stops when Close is called. Interval must be positive for the
// janitor to start.
func WithJanitor(interval, maxAge time.Duration) Option {
	return func(o *runnerOptions) {
		o.janitorInterval = interval
		o.janitorMaxAge = maxAge
	}
}

// NewRunner creates a task runner. By default it uses a concurrency limit of
// 100, derives task contexts from context.Background(), and performs no
// automatic pruning; see WithLimit, WithBaseContext, and WithJanitor.
// Call Close to release the runner's resources.
func NewRunner(opts ...Option) *Runner {
	o := runnerOptions{
		limit:   defaultConcurrencyLimit,
		baseCtx: context.Background(),
	}

	for _, opt := range opts {
		opt(&o)
	}

	var sem chan struct{}
	if o.limit > 0 {
		sem = make(chan struct{}, o.limit)
	}

	ctx, cancel := context.WithCancel(o.baseCtx)

	r := &Runner{
		tasks:  make(map[string]*Task),
		sem:    sem,
		ctx:    ctx,
		cancel: cancel,
	}

	if o.janitorInterval > 0 {
		r.wg.Add(1)

		go r.startJanitor(o.janitorInterval, o.janitorMaxAge)
	}

	return r
}

// Submit queues a function for async execution and returns the task ID.
// The kind parameter identifies the operation type for filtering (e.g. "quality").
//
// The function receives a context derived from the runner's base context;
// it is cancelled when Close is called (or the base context is cancelled)
// and task functions should honour it.
//
// When the concurrency limit is saturated, Submit blocks until a slot frees
// up (backpressure) rather than parking a goroutine per pending task. If the
// runner is closed while waiting, the task is recorded as failed without
// running.
func (r *Runner) Submit(kind string, fn func(ctx context.Context) (any, error)) string {
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

	// Reject deterministically when the runner is already closed. Without
	// this priority check the select below could still pick the semaphore
	// case, because select chooses randomly among ready cases.
	if r.ctx.Err() != nil {
		r.failTask(t, fmt.Errorf("runner closed before task started: %w", r.ctx.Err()))

		return id
	}

	// Acquire the concurrency slot here, not in the spawned goroutine, so a
	// saturated runner applies backpressure to the submitter instead of
	// accumulating one blocked goroutine per pending task.
	if r.sem != nil {
		select {
		case r.sem <- struct{}{}:
		case <-r.ctx.Done():
			r.failTask(t, fmt.Errorf("runner closed before task started: %w", r.ctx.Err()))

			return id
		}
	}

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

// Close cancels the contexts of all outstanding tasks and stops the
// background janitor (if configured), waiting for the janitor to exit.
// Close does not wait for task functions to return; they observe
// cancellation through their context and their results are still recorded.
// Close is idempotent and always returns nil.
func (r *Runner) Close() error {
	r.cancel()
	r.wg.Wait()

	return nil
}

func (r *Runner) startJanitor(interval, maxAge time.Duration) {
	defer r.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.Prune(maxAge)
		}
	}
}

// failTask marks a task as failed without running it.
func (r *Runner) failTask(t *Task, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t.Completed = time.Now()
	t.Status = StatusFailed
	t.Result = Result{Error: err.Error()}
}

func (r *Runner) execute(t *Task, fn func(ctx context.Context) (any, error)) {
	if r.sem != nil {
		// The slot was acquired in Submit; release it when the task finishes.
		defer func() { <-r.sem }()
	}

	// Derive the per-task context from the runner's base context so Close
	// cancels outstanding tasks; cancel on return to release resources.
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

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

	data, err = fn(ctx)

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
