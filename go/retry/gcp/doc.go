// Package gcp provides backward-compatible forwarding shims for [github.com/duizendstra/alexandria/go/platform/retry/gcp].
//
// Deprecated: This package is deprecated. New code should import
// [github.com/duizendstra/alexandria/go/platform/retry/gcp] directly.
//
// # What
//
// An error classification evaluator and execution wrapper forwarding to the canonical
// platform gcp retry package.
//
// # Who
//
// Retained exclusively for backward compatibility with existing downstream consumers.
//
// # When
//
// Migrate all consumer imports to github.com/duizendstra/alexandria/go/platform/retry/gcp.
//
// # Why
//
// Relocated under go/platform/retry/gcp to standardize multi-module taxonomy.
//
// Domain:  Platform
// Concern: Backward-compatible forwarding shims for GCP retry.
package gcp
