// Package finops is the blueprint for the FinOps bounded context.
//
// BC:      FinOps
// Concern: How do we monitor and control cloud spend?
//
// Creates:
//   - FinOps project — hosts the billing dataset and notification channels
//   - BigQuery dataset — receives billing export data
//   - Budget — org-level or folder-level spend alerts
//
// Invariants:
//   - Read-only on billing — observe spend, never modify billing accounts
//   - Alert, don't block — budgets alert humans, never shut down resources
//   - One budget per scope — avoids double-counting
//
// Apply is the composable unit — supports all deployment scenarios:
//
//	Enterprise: func main() { finops.FinOps() }
//	Collapsed:  finops.Apply(ctx, &finops.Params{...}) alongside other BCs
//
// Placement (folder ID, billing account, org ID) resolves in order:
// Params (collapsed mode), then the governance stack reference named by
// the "governanceStack" config key, then explicit stack config.
package finops
