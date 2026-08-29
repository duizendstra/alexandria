// Package retry provides backward-compatible forwarding shims for [github.com/duizendstra/alexandria/go/platform/retry].
//
// Deprecated: This package is deprecated. New code should import
// [github.com/duizendstra/alexandria/go/platform/retry] directly.
//
// # What
//
// A zero-dependency resilience forwarding package pointing to the canonical
// platform resilience module.
//
// # Who
//
// Retained exclusively for backward compatibility with existing downstream consumers.
//
// # When
//
// Migrate all consumer imports to github.com/duizendstra/alexandria/go/platform/retry.
//
// # Why
//
// Relocated under go/platform/retry to achieve 100% taxonomy symmetry with sibling
// platform primitives (such as cache, async, web, and apierr).
//
// Domain:  Platform
// Concern: Backward-compatible forwarding shims for resilience and retry.
package retry
