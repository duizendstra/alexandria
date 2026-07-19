// Package scope defines the organizational level at which governance operates.
//
// BC:      Governance
// Concern: What capabilities are available at each operational scope?
//
// This package is pure Go — zero external dependencies. It models the
// governance policy that certain capabilities (tag key management,
// org-level exports) are only available at organization scope,
// while hierarchy management works at any scope.
//
// Cloud-agnostic — adapters translate cloud-specific parent formats
// into a Scope value. The domain only reasons about capabilities.
package scope
