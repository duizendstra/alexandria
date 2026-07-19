// Package governance is the blueprint for the governance bounded context.
//
// BC:      Governance
// Concern: How is governance realized on the platform?
//
// This blueprint reads Pulumi configuration, builds a domain Plan,
// validates it, then delegates to platform adapters (GCP folders,
// GCP tag keys) and exports the results for downstream BCs.
//
// The domain logic (what capabilities exist at each scope, what gets
// exported) lives in the Plan and Scope packages. This blueprint only
// handles the mechanical translation: config → Plan → adapters → exports.
//
// Enterprise customer (separate programs):
//
//	func main() { governance.Governance() }
//
// Small customer (all BCs in one program):
//
//	func main() {
//	    pulumi.Run(func(ctx *pulumi.Context) error {
//	        if err := governance.Apply(ctx); err != nil { return err }
//	        if err := identity.Apply(ctx); err != nil { return err }
//	        return nil
//	    })
//	}
package governance
