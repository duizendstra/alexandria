package workflow_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/workflow"
)

func TestWorkflow_SequentialExecution(t *testing.T) {
	t.Parallel()

	var sequence []string
	w := workflow.New("deploy-pipeline")
	w.Add("step-1", func(_ context.Context) error {
		sequence = append(sequence, "1")
		return nil
	}).Add("step-2", func(_ context.Context) error {
		sequence = append(sequence, "2")
		return nil
	})

	err := w.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sequence) != 2 || sequence[0] != "1" || sequence[1] != "2" {
		t.Fatalf("unexpected sequence: %v", sequence)
	}
}

func TestWorkflow_PanicRecovery(t *testing.T) {
	t.Parallel()

	w := workflow.New("panic-flow")
	w.Add("safe-step", func(_ context.Context) error {
		return nil
	}).Add("panicking-step", func(_ context.Context) error {
		panic("nil pointer dereference simulation")
	}).Add("unreached-step", func(_ context.Context) error {
		return nil
	})

	res := w.RunWithResult(context.Background())
	if res.Err == nil {
		t.Fatal("expected workflow to return error on panic, got nil")
	}

	if !errors.Is(res.Err, workflow.ErrPanicRecovered) {
		t.Fatalf("expected ErrPanicRecovered, got: %v", res.Err)
	}

	if len(res.StepResults) != 3 {
		t.Fatalf("expected 3 step results, got %d", len(res.StepResults))
	}

	if res.StepResults[0].Status != workflow.StatusSuccess {
		t.Fatalf("step 0 status: got %s, want %s", res.StepResults[0].Status, workflow.StatusSuccess)
	}
	if res.StepResults[1].Status != workflow.StatusFailed {
		t.Fatalf("step 1 status: got %s, want %s", res.StepResults[1].Status, workflow.StatusFailed)
	}
	if res.StepResults[2].Status != workflow.StatusNotRun {
		t.Fatalf("step 2 status: got %s, want %s", res.StepResults[2].Status, workflow.StatusNotRun)
	}
}

func TestWorkflow_NilRunFunction(t *testing.T) {
	t.Parallel()

	w := workflow.New("nil-run")
	w.AddStep(workflow.Step{Name: "nil-step", Run: nil})

	err := w.Run(context.Background())
	if err == nil {
		t.Fatal("expected error on nil step run, got nil")
	}
	if !errors.Is(err, workflow.ErrNilRun) {
		t.Fatalf("expected ErrNilRun, got: %v", err)
	}
}

func TestWorkflow_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var step1Run, step2Run bool

	w := workflow.New("cancellable-flow")
	w.Add("step-1", func(_ context.Context) error {
		step1Run = true
		cancel() // cancel context inside step 1.
		return nil
	}).Add("step-2", func(_ context.Context) error {
		step2Run = true
		return nil
	})

	res := w.RunWithResult(ctx)
	if res.Err == nil {
		t.Fatal("expected context canceled error, got nil")
	}
	if !errors.Is(res.Err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", res.Err)
	}

	if !step1Run {
		t.Error("step 1 should have run")
	}
	if step2Run {
		t.Error("step 2 should not have run after cancellation")
	}

	if len(res.StepResults) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(res.StepResults))
	}
	if res.StepResults[1].Status != workflow.StatusNotRun {
		t.Errorf("step 2 status: got %s, want %s", res.StepResults[1].Status, workflow.StatusNotRun)
	}
}

func TestWorkflow_StepSkip(t *testing.T) {
	t.Parallel()

	var step2Executed bool
	w := workflow.New("skip-flow")
	w.Add("step-1", func(_ context.Context) error { return nil })
	w.AddStep(workflow.Step{
		Name: "step-2",
		Skip: func(_ context.Context) bool { return true },
		Run: func(_ context.Context) error {
			step2Executed = true
			return nil
		},
	})
	w.Add("step-3", func(_ context.Context) error { return nil })

	res := w.RunWithResult(context.Background())
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}

	if step2Executed {
		t.Error("step 2 should have been skipped")
	}

	if len(res.StepResults) != 3 {
		t.Fatalf("expected 3 step results, got %d", len(res.StepResults))
	}
	if res.StepResults[1].Status != workflow.StatusSkipped {
		t.Errorf("step 2 status: got %s, want %s", res.StepResults[1].Status, workflow.StatusSkipped)
	}
}

func TestWorkflow_DryRun(t *testing.T) {
	t.Parallel()

	var executed bool
	w := workflow.New("dry-run-flow")
	w.Add("step-1", func(_ context.Context) error {
		executed = true
		return nil
	})
	w.AddStep(workflow.Step{
		Name: "step-2",
		Skip: func(_ context.Context) bool { return true },
		Run: func(_ context.Context) error {
			executed = true
			return nil
		},
	})

	res := w.DryRun(context.Background())
	if executed {
		t.Error("DryRun must not execute Step.Run functions")
	}

	if len(res.StepResults) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(res.StepResults))
	}
	if res.StepResults[0].Status != workflow.StatusSuccess {
		t.Errorf("step 1 dry-run status: got %s, want %s", res.StepResults[0].Status, workflow.StatusSuccess)
	}
	if res.StepResults[1].Status != workflow.StatusSkipped {
		t.Errorf("step 2 dry-run status: got %s, want %s", res.StepResults[1].Status, workflow.StatusSkipped)
	}
}

func TestWorkflow_LifecycleHooks(t *testing.T) {
	t.Parallel()

	var (
		workflowStarted atomic.Bool
		workflowDone    atomic.Bool
		stepsStarted    atomic.Int64
		stepsDone       atomic.Int64
		stepsFailed     atomic.Int64
		stepsSkipped    atomic.Int64
	)

	errDummy := errors.New("dummy step failure")

	hooks := workflow.Hooks{
		OnWorkflowStart: func(_ context.Context, name string, count int) {
			if name == "hook-flow" && count == 3 {
				workflowStarted.Store(true)
			}
		},
		OnStepStart: func(_ context.Context, _, _ int, _ workflow.Step) {
			stepsStarted.Add(1)
		},
		OnStepDone: func(_ context.Context, _, _ int, _ workflow.Step, _ time.Duration) {
			stepsDone.Add(1)
		},
		OnStepFail: func(_ context.Context, _, _ int, _ workflow.Step, _ error, _ time.Duration) {
			stepsFailed.Add(1)
		},
		OnStepSkip: func(_ context.Context, _, _ int, _ workflow.Step) {
			stepsSkipped.Add(1)
		},
		OnWorkflowDone: func(_ context.Context, _ string, _ time.Duration, _ error) {
			workflowDone.Store(true)
		},
	}

	w := workflow.New("hook-flow", workflow.WithHooks(hooks))
	w.Add("step-1", func(_ context.Context) error { return nil })
	w.AddStep(workflow.Step{
		Name: "step-2-skip",
		Skip: func(_ context.Context) bool { return true },
		Run:  func(_ context.Context) error { return nil },
	})
	w.Add("step-3-fail", func(_ context.Context) error {
		return errDummy
	})

	err := w.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !workflowStarted.Load() {
		t.Error("OnWorkflowStart hook did not trigger")
	}
	if !workflowDone.Load() {
		t.Error("OnWorkflowDone hook did not trigger")
	}
	if stepsStarted.Load() != 2 {
		t.Errorf("expected 2 steps started, got %d", stepsStarted.Load())
	}
	if stepsDone.Load() != 1 {
		t.Errorf("expected 1 step done, got %d", stepsDone.Load())
	}
	if stepsSkipped.Load() != 1 {
		t.Errorf("expected 1 step skipped, got %d", stepsSkipped.Load())
	}
	if stepsFailed.Load() != 1 {
		t.Errorf("expected 1 step failed, got %d", stepsFailed.Load())
	}
}

func FuzzWorkflowExecution(f *testing.F) {
	f.Add(3, 1, false)
	f.Add(5, 0, true)
	f.Add(10, 8, false)

	errFuzz := errors.New("fuzz failure")

	f.Fuzz(func(t *testing.T, stepCount int, failAtStep int, cancelContext bool) {
		if stepCount <= 0 || stepCount > 50 {
			return
		}

		ctx := context.Background()
		var cancel context.CancelFunc
		if cancelContext {
			ctx, cancel = context.WithCancel(ctx)
			defer cancel()
		}

		w := workflow.New("fuzz-flow")
		for i := range stepCount {
			stepIdx := i
			w.AddStep(workflow.Step{
				Name: "step",
				Run: func(_ context.Context) error {
					if cancelContext && stepIdx == failAtStep {
						cancel()
					}
					if stepIdx == failAtStep && !cancelContext {
						return errFuzz
					}
					return nil
				},
			})
		}

		res := w.RunWithResult(ctx)
		if failAtStep >= 0 && failAtStep < stepCount {
			if res.Err == nil && !cancelContext {
				t.Fatalf("expected error at step %d, got nil", failAtStep)
			}
		}
	})
}
