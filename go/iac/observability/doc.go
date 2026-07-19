// Package observability is the blueprint for the Observability bounded
// context.
//
// BC:      Observability
// Concern: What is happening across the cloud estate?
//
// Creates:
//   - Observability project — hosts the log analytics dataset
//   - BigQuery dataset — receives aggregated audit/activity logs
//   - Org-level log sink — routes audit logs to the dataset
//
// The sink's writer identity is exported so downstream stacks can grant
// it BigQuery access.
//
// Apply is the composable unit — supports all deployment scenarios:
//
//	Enterprise: func main() { observability.Observability() }
//	Collapsed:  observability.Apply(ctx, &observability.Params{...}) alongside other BCs
//
// Placement (folder ID, billing account, org ID) resolves in order:
// Params (collapsed mode), then the governance stack reference named by
// the "governanceStack" config key (folder chosen by the optional
// "governanceFolder" key, default "shared"), then explicit stack config.
package observability
