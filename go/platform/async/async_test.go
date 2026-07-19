package async_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/async"
)

// waitForStatus polls until the task reaches the wanted status or the timeout expires.
func waitForStatus(t *testing.T, r *async.Runner, id string, want async.Status) *async.Task {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if task := r.Get(id); task != nil && task.Status == want {
			return task
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("task %s did not reach status %s in time", id, want)

	return nil
}

func TestSubmitAndGet(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id := r.Submit("quality", func(_ context.Context) (any, error) {
		return "all passed", nil
	})

	if id == "" {
		t.Fatal("expected non-empty task ID")
	}

	task := waitForStatus(t, r, id, async.StatusDone)

	if task.Result.Data != "all passed" {
		t.Errorf("expected result 'all passed', got %v", task.Result.Data)
	}

	if task.Result.Error != "" {
		t.Errorf("expected no error, got %s", task.Result.Error)
	}

	if task.Kind != "quality" {
		t.Errorf("expected kind 'quality', got %s", task.Kind)
	}
}

func TestSubmitFailedTask(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id := r.Submit("build", func(_ context.Context) (any, error) {
		return nil, errors.New("lint failed")
	})

	task := waitForStatus(t, r, id, async.StatusFailed)

	if task.Result.Error != "lint failed" {
		t.Errorf("expected error 'lint failed', got %s", task.Result.Error)
	}
}

func TestGetUnknownTask(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	if task := r.Get("nonexistent"); task != nil {
		t.Errorf("expected nil for unknown task, got %+v", task)
	}
}

func TestList(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id1 := r.Submit("quality", func(_ context.Context) (any, error) { return "", nil })
	id2 := r.Submit("build", func(_ context.Context) (any, error) { return "", nil })

	waitForStatus(t, r, id1, async.StatusDone)
	waitForStatus(t, r, id2, async.StatusDone)

	tasks := r.List()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestPrune(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id := r.Submit("old", func(_ context.Context) (any, error) { return "", nil })

	waitForStatus(t, r, id, async.StatusDone)

	// Prune with zero age — should remove all completed tasks.
	removed := r.Prune(0)
	if removed != 1 {
		t.Errorf("expected 1 pruned, got %d", removed)
	}

	if len(r.List()) != 0 {
		t.Errorf("expected 0 tasks after prune, got %d", len(r.List()))
	}
}

func TestGetReturnsCopy(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id := r.Submit("test", func(_ context.Context) (any, error) { return "ok", nil })

	waitForStatus(t, r, id, async.StatusDone)

	t1 := r.Get(id)
	t2 := r.Get(id)

	if t1 == t2 {
		t.Error("Get should return independent copies, not the same pointer")
	}
}

func TestTaskPanicRecovery(t *testing.T) {
	r := async.NewRunner()
	defer func() { _ = r.Close() }()

	id := r.Submit("panic-test", func(_ context.Context) (any, error) {
		panic("boom!")
	})

	task := waitForStatus(t, r, id, async.StatusFailed)

	if task.Result.Error != "panic: boom!" {
		t.Errorf("expected error message 'panic: boom!', got %q", task.Result.Error)
	}
}

func TestConcurrencyLimitAppliesBackpressure(t *testing.T) {
	// Setup with limit of 1.
	r := async.NewRunner(async.WithLimit(1))
	defer func() { _ = r.Close() }()

	started := make(chan struct{})
	block := make(chan struct{})

	// Task 1: blocks until we release it.
	r.Submit("blocker", func(_ context.Context) (any, error) {
		close(started)
		<-block

		return "done1", nil
	})

	<-started

	// Task 2: Submit must block (backpressure) while the blocker holds the
	// only concurrency slot, instead of spawning a parked goroutine.
	submitted := make(chan string)

	go func() {
		submitted <- r.Submit("queued", func(_ context.Context) (any, error) {
			return "done2", nil
		})
	}()

	select {
	case <-submitted:
		t.Fatal("Submit should block while the concurrency limit is saturated")
	case <-time.After(50 * time.Millisecond):
		// Expected: Submit is blocked.
	}

	// Release Task 1; Task 2 should now be admitted and complete.
	close(block)

	var id2 string
	select {
	case id2 = <-submitted:
	case <-time.After(2 * time.Second):
		t.Fatal("Submit did not unblock after slot was released")
	}

	t2 := waitForStatus(t, r, id2, async.StatusDone)
	if t2.Result.Data != "done2" {
		t.Errorf("expected 'done2', got %v", t2.Result.Data)
	}
}

// TestCloseCancelsTaskContexts verifies that the runner's Close cancels the
// context handed to running tasks.
func TestCloseCancelsTaskContexts(t *testing.T) {
	r := async.NewRunner()

	started := make(chan struct{})

	id := r.Submit("cancellable", func(ctx context.Context) (any, error) {
		close(started)
		<-ctx.Done()

		return nil, ctx.Err()
	})

	<-started

	if err := r.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	task := waitForStatus(t, r, id, async.StatusFailed)

	if task.Result.Error != context.Canceled.Error() {
		t.Errorf("expected error %q, got %q", context.Canceled.Error(), task.Result.Error)
	}
}

// TestBaseContextCancellationPropagates verifies that cancelling the base
// context supplied via WithBaseContext cancels per-task contexts.
func TestBaseContextCancellationPropagates(t *testing.T) {
	baseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := async.NewRunner(async.WithBaseContext(baseCtx))
	defer func() { _ = r.Close() }()

	started := make(chan struct{})

	id := r.Submit("cancellable", func(ctx context.Context) (any, error) {
		close(started)
		<-ctx.Done()

		return nil, ctx.Err()
	})

	<-started
	cancel()

	task := waitForStatus(t, r, id, async.StatusFailed)

	if task.Result.Error != context.Canceled.Error() {
		t.Errorf("expected error %q, got %q", context.Canceled.Error(), task.Result.Error)
	}
}

// TestSubmitAfterCloseFailsTask verifies that submitting to a closed runner
// records the task as failed instead of running it.
func TestSubmitAfterCloseFailsTask(t *testing.T) {
	r := async.NewRunner()

	if err := r.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	id := r.Submit("late", func(_ context.Context) (any, error) {
		t.Error("task function must not run after Close")

		return nil, nil
	})

	task := waitForStatus(t, r, id, async.StatusFailed)

	if task.Result.Error == "" {
		t.Error("expected a non-empty error for task submitted after Close")
	}
}

// TestGoroutinesBoundedUnderLoad is the regression test for unbounded
// goroutine growth: previously each Submit spawned a goroutine that parked on
// the semaphore, so N pending tasks meant N goroutines. Now the slot is
// acquired in Submit, bounding spawned goroutines to the concurrency limit.
func TestGoroutinesBoundedUnderLoad(t *testing.T) {
	const (
		limit     = 4
		submitted = 50
		tolerance = 6
	)

	r := async.NewRunner(async.WithLimit(limit))
	defer func() { _ = r.Close() }()

	gate := make(chan struct{})
	baseline := runtime.NumGoroutine()

	done := make(chan struct{})

	go func() {
		defer close(done)

		for range submitted {
			r.Submit("slow", func(ctx context.Context) (any, error) {
				select {
				case <-gate:
				case <-ctx.Done():
				}

				return nil, nil
			})
		}
	}()

	// Give the submitter time to saturate the limit.
	time.Sleep(100 * time.Millisecond)

	// One submitter goroutine + at most `limit` task goroutines may be live.
	if n := runtime.NumGoroutine(); n > baseline+limit+tolerance {
		t.Errorf("goroutine count not bounded: baseline %d, limit %d, got %d", baseline, limit, n)
	}

	close(gate)
	<-done

	// All tasks must still complete.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tasks := r.List()

		doneCount := 0
		for _, task := range tasks {
			if task.Status == async.StatusDone {
				doneCount++
			}
		}

		if doneCount == submitted {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("not all tasks completed: %d done", len(r.List()))
}

// TestJanitorPrunesCompletedTasks verifies that the optional background
// janitor removes completed tasks without manual Prune calls.
func TestJanitorPrunesCompletedTasks(t *testing.T) {
	r := async.NewRunner(async.WithJanitor(10*time.Millisecond, 0))
	defer func() { _ = r.Close() }()

	id := r.Submit("quick", func(_ context.Context) (any, error) { return "ok", nil })

	waitForStatus(t, r, id, async.StatusDone)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r.Get(id) == nil {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("janitor did not prune the completed task in time")
}

// TestCloseIsIdempotent verifies Close can be called multiple times safely.
func TestCloseIsIdempotent(t *testing.T) {
	r := async.NewRunner(async.WithJanitor(10*time.Millisecond, time.Minute))

	if err := r.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}
