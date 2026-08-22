package workflow_test

import (
	"context"
	"fmt"

	"github.com/duizendstra/alexandria/go/platform/workflow"
)

func ExampleWorkflow_Run() {
	ctx := context.Background()

	wf := workflow.New("Setup Service").
		Add("Step 1: Check Config", func(ctx context.Context) error {
			fmt.Println("Config valid")
			return nil
		}).
		Add("Step 2: Init Store", func(ctx context.Context) error {
			fmt.Println("Store initialized")
			return nil
		})

	if err := wf.Run(ctx); err != nil {
		fmt.Printf("failed: %v\n", err)
	}

	// Output:
	// Config valid
	// Store initialized
}

func ExampleWorkflow_AddStep() {
	ctx := context.Background()
	featureEnabled := false

	wf := workflow.New("Deploy Feature").
		AddStep(workflow.Step{
			Name: "Optional Feature Setup",
			Run: func(ctx context.Context) error {
				fmt.Println("Feature setup executed")
				return nil
			},
			Skip: func(ctx context.Context) bool {
				return !featureEnabled
			},
		}).
		Add("Core Setup", func(ctx context.Context) error {
			fmt.Println("Core setup executed")
			return nil
		})

	if err := wf.Run(ctx); err != nil {
		fmt.Printf("failed: %v\n", err)
	}

	// Output:
	// Core setup executed
}
