// Package plan defines the complete desired governance state.
//
// BC:      Governance
// Concern: What is the full specification of what governance should produce?
//
// A Plan is a pure value object — it describes the desired state
// without knowing how to achieve it. Tools (Pulumi, Terraform, etc.)
// consume a Plan and produce infrastructure. The Plan is the
// abstraction boundary between domain knowledge and tool mechanics.
//
// This package depends only on other governance domain packages
// (hierarchy, classification, scope). Zero tool or cloud dependencies.
package plan
