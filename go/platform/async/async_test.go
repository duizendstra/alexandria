package async_test

import (
	"errors"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/async"
)

func TestSubmitAndGet(t *testing.T) {
	r := async.NewRunner()

	id := r.Submit("quality", func() (any, error) {
		return "all passed", nil
	})

	if id == "" {
		t.Fatal("expected non-empty task ID")
	}

	// Wait for completion.
	var task *async.Task

	for range 50 {
		task = r.Get(id)
		if task != nil && task.Status == async.StatusDone {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if task == nil {
		t.Fatal("task not found")
	}

	if task.Status != async.StatusDone {
		t.Errorf("expected StatusDone, got %s", task.Status)
	}

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

	id := r.Submit("build", func() (any, error) {
		return nil, errors.New("lint failed")
	})

	var task *async.Task

	for range 50 {
		task = r.Get(id)
		if task != nil && task.Status == async.StatusFailed {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if task == nil {
		t.Fatal("task not found")
	}

	if task.Status != async.StatusFailed {
		t.Errorf("expected StatusFailed, got %s", task.Status)
	}

	if task.Result.Error != "lint failed" {
		t.Errorf("expected error 'lint failed', got %s", task.Result.Error)
	}
}

func TestGetUnknownTask(t *testing.T) {
	r := async.NewRunner()

	if task := r.Get("nonexistent"); task != nil {
		t.Errorf("expected nil for unknown task, got %+v", task)
	}
}

func TestList(t *testing.T) {
	r := async.NewRunner()

	r.Submit("quality", func() (any, error) { return "", nil })
	r.Submit("build", func() (any, error) { return "", nil })

	// Wait for both to complete.
	time.Sleep(50 * time.Millisecond)

	tasks := r.List()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestPrune(t *testing.T) {
	r := async.NewRunner()

	r.Submit("old", func() (any, error) { return "", nil })

	// Wait for completion.
	time.Sleep(50 * time.Millisecond)

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

	id := r.Submit("test", func() (any, error) { return "ok", nil })

	time.Sleep(50 * time.Millisecond)

	t1 := r.Get(id)
	t2 := r.Get(id)

	if t1 == t2 {
		t.Error("Get should return independent copies, not the same pointer")
	}
}

func TestTaskPanicRecovery(t *testing.T) {
	r := async.NewRunner()

	id := r.Submit("panic-test", func() (any, error) {
		panic("boom!")
	})

	time.Sleep(50 * time.Millisecond)

	task := r.Get(id)
	if task == nil {
		t.Fatal("expected task to exist")
	}

	if task.Status != async.StatusFailed {
		t.Errorf("expected StatusFailed, got %s", task.Status)
	}

	if task.Result.Error != "panic: boom!" {
		t.Errorf("expected error message 'panic: boom!', got %q", task.Result.Error)
	}
}

func TestConcurrencyLimit(t *testing.T) {
	// Setup with limit of 1
	r := async.NewRunnerWithLimit(1)

	started := make(chan struct{})
	block := make(chan struct{})

	// Task 1: blocks until we release it
	r.Submit("blocker", func() (any, error) {
		close(started)
		<-block
		return "done1", nil
	})

	<-started

	// Task 2: submitted but should not run because limit of 1 is held by blocker
	id2 := r.Submit("queued", func() (any, error) {
		return "done2", nil
	})

	time.Sleep(20 * time.Millisecond)

	t2 := r.Get(id2)
	if t2 == nil {
		t.Fatal("expected task 2 to exist")
	}
	if t2.Status != async.StatusPending {
		t.Errorf("expected Task 2 to be StatusPending, got %s", t2.Status)
	}

	// Release Task 1
	close(block)

	time.Sleep(50 * time.Millisecond)

	t2 = r.Get(id2)
	if t2.Status != async.StatusDone {
		t.Errorf("expected Task 2 to complete, got %s", t2.Status)
	}
}
