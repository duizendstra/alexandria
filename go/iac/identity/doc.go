// Package identity is the blueprint for the identity bounded context.
//
// BC:      Identity
// Concern: How do we manage credentials, service accounts, and access?
//
// Apply is the composable unit — supports all deployment scenarios:
//
//	Enterprise: func main() { identity.Identity() }
//	Collapsed:  identity.Apply(ctx, &identity.Params{...}) alongside other BCs
//
// Scope: works at folder level (creates its own project in a folder).
// Secret values must be provided at deploy time via a SecretResolver
// (default: the local pass store).
//
// Placement (folder ID and billing account) resolves in order: Params
// (collapsed mode), then the governance stack reference named by the
// "governanceStack" config key, then explicit stack config.
package identity
