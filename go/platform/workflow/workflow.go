package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for workflow failures.
var (
	// ErrNilRun is returned when a Step is executed with a nil Run function.
	ErrNilRun = errors.New("nil step run function")

	// ErrPanicRecovered is wrapped when a Step's Run function panics.
	ErrPanicRecovered = errors.New("panic recovered")
)

// StepStatus represents the terminal state of a Step in a Workflow run.
type StepStatus string

const (
	// StatusSuccess indicates the step ran and returned nil.
	StatusSuccess StepStatus = "success"
	// StatusSkipped indicates the step was skipped by its Skip predicate.
	StatusSkipped StepStatus = "skipped"
	// StatusFailed indicates the step returned an error or panicked.
	StatusFailed StepStatus = "failed"
	// StatusNotRun indicates the workflow was stopped before reaching this step.
	StatusNotRun StepStatus = "not_run"
)

// Step represents a single executable unit inside a larger workflow.
type Step struct {
	// Name is a short identifier for the step.
	Name string
	// Description optionally explains what the step accomplishes.
	Description string
	// Run is the execution body of the step.
	Run func(ctx context.Context) error
	// Skip is an optional predicate; if it returns true, Run is skipped.
	Skip func(ctx context.Context) bool
}

// Hooks defines callbacks for lifecycle events during a Workflow run.
type Hooks struct {
	// OnWorkflowStart is called before executing the first step.
	OnWorkflowStart func(ctx context.Context, name string, stepCount int)
	// OnStepStart is called immediately before a step's Run function executes.
	OnStepStart func(ctx context.Context, index, total int, step Step)
	// OnStepDone is called after a step successfully executes without error.
	OnStepDone func(ctx context.Context, index, total int, step Step, duration time.Duration)
	// OnStepFail is called when a step returns an error or panics.
	OnStepFail func(ctx context.Context, index, total int, step Step, err error, duration time.Duration)
	// OnStepSkip is called when a step is skipped because its Skip predicate returned true.
	OnStepSkip func(ctx context.Context, index, total int, step Step)
	// OnWorkflowDone is called after all steps have completed or after a failure halts execution.
	OnWorkflowDone func(ctx context.Context, name string, duration time.Duration, err error)
}

// StepResult captures the outcome of an individual Step.
type StepResult struct {
	Step     Step
	Status   StepStatus
	Duration time.Duration
	Err      error
}

// Result captures the aggregated execution results of a complete Workflow.
type Result struct {
	Name        string
	Duration    time.Duration
	StepResults []StepResult
	Err         error
}

// Option configures a Workflow.
type Option func(*Workflow)

// WithHooks sets the lifecycle hooks for the Workflow.
func WithHooks(hooks Hooks) Option {
	return func(w *Workflow) {
		w.hooks = hooks
	}
}

// Workflow orchestrates and executes an ordered series of sequential Steps.
type Workflow struct {
	Name  string
	Steps []Step
	hooks Hooks
}

// New creates a new named Workflow.
func New(name string, opts ...Option) *Workflow {
	w := &Workflow{
		Name: name,
	}
	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Add appends a named step with a run function to the workflow.
func (w *Workflow) Add(name string, run func(ctx context.Context) error) *Workflow {
	return w.AddStep(Step{
		Name: name,
		Run:  run,
	})
}

// AddStep appends a configured Step to the workflow.
func (w *Workflow) AddStep(step Step) *Workflow {
	w.Steps = append(w.Steps, step)

	return w
}

// Run executes all steps in order. It respects context cancellation before and
// during each step, recovers gracefully from step panics, invokes configured
// lifecycle hooks, and stops execution immediately upon the first step error.
func (w *Workflow) Run(ctx context.Context) error {
	res := w.RunWithResult(ctx)

	return res.Err
}

// RunWithResult executes all steps and returns a structured Result detailing
// the status and timing of each step.
func (w *Workflow) RunWithResult(ctx context.Context) Result {
	startTime := time.Now()
	total := len(w.Steps)
	res := Result{
		Name:        w.Name,
		StepResults: make([]StepResult, 0, total),
	}

	if w.hooks.OnWorkflowStart != nil {
		w.hooks.OnWorkflowStart(ctx, w.Name, total)
	}

	for i, step := range w.Steps {
		// Check context cancellation before starting the step.
		if ctxErr := ctx.Err(); ctxErr != nil {
			err := fmt.Errorf("workflow halted: context canceled: %w", ctxErr)
			res.Err = err
			// Mark remaining steps as not run.
			for j := i; j < total; j++ {
				res.StepResults = append(res.StepResults, StepResult{
					Step:   w.Steps[j],
					Status: StatusNotRun,
				})
			}

			break
		}

		// Check skip predicate.
		if step.Skip != nil && step.Skip(ctx) {
			if w.hooks.OnStepSkip != nil {
				w.hooks.OnStepSkip(ctx, i+1, total, step)
			}
			res.StepResults = append(res.StepResults, StepResult{
				Step:   step,
				Status: StatusSkipped,
			})

			continue
		}

		if w.hooks.OnStepStart != nil {
			w.hooks.OnStepStart(ctx, i+1, total, step)
		}

		stepStart := time.Now()
		stepErr := runStepWithRecovery(ctx, step.Run)
		stepDuration := time.Since(stepStart)

		if stepErr != nil {
			wrappedErr := fmt.Errorf("step '%s' failed: %w", step.Name, stepErr)
			if w.hooks.OnStepFail != nil {
				w.hooks.OnStepFail(ctx, i+1, total, step, stepErr, stepDuration)
			}
			res.StepResults = append(res.StepResults, StepResult{
				Step:     step,
				Status:   StatusFailed,
				Duration: stepDuration,
				Err:      wrappedErr,
			})
			res.Err = wrappedErr

			// Mark remaining steps as not run.
			for j := i + 1; j < total; j++ {
				res.StepResults = append(res.StepResults, StepResult{
					Step:   w.Steps[j],
					Status: StatusNotRun,
				})
			}

			break
		}

		if w.hooks.OnStepDone != nil {
			w.hooks.OnStepDone(ctx, i+1, total, step, stepDuration)
		}
		res.StepResults = append(res.StepResults, StepResult{
			Step:     step,
			Status:   StatusSuccess,
			Duration: stepDuration,
		})
	}

	res.Duration = time.Since(startTime)
	if w.hooks.OnWorkflowDone != nil {
		w.hooks.OnWorkflowDone(ctx, w.Name, res.Duration, res.Err)
	}

	return res
}

// DryRun simulates the workflow execution without invoking the step Run bodies,
// evaluating Skip predicates and producing a planned Result.
func (w *Workflow) DryRun(ctx context.Context) Result {
	total := len(w.Steps)
	res := Result{
		Name:        w.Name,
		StepResults: make([]StepResult, 0, total),
	}

	for _, step := range w.Steps {
		if step.Skip != nil && step.Skip(ctx) {
			res.StepResults = append(res.StepResults, StepResult{
				Step:   step,
				Status: StatusSkipped,
			})
		} else {
			res.StepResults = append(res.StepResults, StepResult{
				Step:   step,
				Status: StatusSuccess,
			})
		}
	}

	return res
}

func runStepWithRecovery(ctx context.Context, run func(context.Context) error) (err error) {
	if run == nil {
		return ErrNilRun
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrPanicRecovered, r)
		}
	}()

	return run(ctx)
}
