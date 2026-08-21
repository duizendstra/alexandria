// Package workflow provides a standard-library-first, context-aware sequential
// step runner designed for multi-step CLI procedures, setup wizards, rollout
// chains, and database or infrastructure provisioning workflows.
//
// Workflows execute an ordered sequence of named Steps. Features include:
//   - Context cancellation checks before and during step execution
//   - Per-step panic recovery with error wrapping and stack safety
//   - Conditional step skipping via Skip predicates
//   - Configurable lifecycle hooks (OnStepStart, OnStepDone, OnStepFail, OnStepSkip)
//   - Dry-run execution mode
//   - Detailed step-by-step summary reporting
package workflow
