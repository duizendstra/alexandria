// Package delivery is the blueprint for the Delivery bounded context.
//
// BC:      Delivery
// Concern: How do we build and deploy code?
//
// Creates:
//   - Delivery project — hosts CI/CD resources
//   - Artifact Registry — container image storage
//   - Git hosting connection — authenticates builds to the provider
//   - Cloud Build triggers — per-repo, per-environment
//   - Cross-project registry reader grants for consumer workloads
//
// All customer-specific values (registry ID, repos, triggers,
// consumer stacks) come from Pulumi stack config — no hardcoded names.
//
// Apply is the composable unit — supports all deployment scenarios:
//
//	Enterprise: func main() { delivery.Delivery() }
//	Collapsed:  delivery.Apply(ctx, &delivery.Params{...}) alongside other BCs
//
// Placement (folder ID and billing account) resolves in order: Params
// (collapsed mode), then the governance stack reference named by the
// "governanceStack" config key (folder chosen by the optional
// "governanceFolder" key, default "shared"), then explicit stack config.
package delivery
