// Package apierr defines sentinel errors for vendor API interactions.
//
// # What
//
// A set of sentinel errors covering HTTP/gRPC failure modes and a
// [StatusError] type for rich error context. Sentinels use semantic
// names that map cleanly to both HTTP status codes and gRPC status codes.
//
// # Who
//
// Vendor client authors return these errors. Consumers match them with
// [errors.Is] for category and [errors.As] with [StatusError] for context.
// The [IsRetryable] helper simplifies retry decisions.
//
// # When
//
// Use apierr errors whenever a vendor API call fails. Never define ad-hoc
// error values in vendor packages — wrap apierr sentinels instead:
//
//	var ErrRateLimited = fmt.Errorf("postmark: %w", apierr.ErrRateLimited)
//
// # Where
//
// Sits at the bottom of the dependency graph — zero internal dependencies.
// Consumed by vendor clients via [retry.Transport] and [auth.Transport].
//
// # Why
//
// Centralizing errors makes the failure surface explicit. Consumers write
// one retry/fallback implementation that works for every vendor.
//
// Domain:  Platform
// Concern: How do we represent API errors consistently?
package apierr
