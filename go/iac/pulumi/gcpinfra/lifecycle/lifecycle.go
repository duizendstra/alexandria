// Package lifecycle carries the one decision every building block in this
// module needs before it creates anything: is this stack permanent, or is it
// disposable?
//
// A Pulumi logical-name change is a delete plus a create, not an update. On a
// data-bearing resource that is data loss, so the blocks mark those resources
// pulumi.Protect(true) by default and the engine refuses the delete at preview
// instead of at apply. Throwaway stacks pass Ephemeral to opt out, which is
// what makes `pulumi destroy` work on them at all.
//
// Protect lives in stack state and is written by an up, so it cannot guard the
// very update that introduces it: an up that both adds protection and renames
// still deletes the old resource, because the state it consults was written
// before the protection existed. Deploy the protection first, rename after.
package lifecycle

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// Option configures how a building block treats the resources it creates.
type Option func(*settings)

type settings struct {
	ephemeral bool
}

// Ephemeral marks the stack disposable: its data-bearing resources are created
// unprotected, at both the Pulumi and the GCP layer, so a destroy tears them
// down without a manual `pulumi state unprotect` first. Pass it only for test
// and preview stacks — it is the single opt-out from the protection every
// other stack gets by default.
func Ephemeral() Option {
	return func(s *settings) { s.ephemeral = true }
}

// IsEphemeral reports whether opts mark the stack disposable. Building blocks
// use it to derive the GCP-level deletion flags that sit underneath Protect.
func IsEphemeral(opts ...Option) bool {
	var s settings

	for _, opt := range opts {
		if opt != nil {
			opt(&s)
		}
	}

	return s.ephemeral
}

// Protect returns the resource option that guards a data-bearing resource.
//
// The false case is deliberately explicit rather than omitted: Pulumi treats
// an unset protect as "inherit", and an ephemeral stack needs the flag cleared
// on resources a previous up protected, not left to inheritance.
func Protect(opts ...Option) pulumi.ResourceOption {
	return pulumi.Protect(!IsEphemeral(opts...))
}
