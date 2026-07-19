// Package workloads is the blueprint for the workloads bounded context.
//
// BC:      Workloads
// Concern: How do we provision multi-project workload environments?
//
// An environment consists of one or more projects, each serving one or
// more concerns (e.g. compute, data, reports). In development all
// concerns typically collapse into a single project; in acceptance and
// production each concern gets its own project with dedicated APIs.
//
// Apply is the composable unit — supports all deployment scenarios:
//
//	Enterprise: func main() { workloads.Workloads() }
//	Collapsed:  workloads.Apply(ctx, &workloads.Params{...}) alongside other BCs
//
// Placement (folder ID and billing account) resolves in order: Params
// (collapsed mode), then the governance stack reference named by the
// "governanceStack" config key (the folder is chosen by the required
// "environmentFolder" key), then explicit stack config.
//
// Exports per concern: {concern}ProjectId, {concern}ProjectNumber.
package workloads
