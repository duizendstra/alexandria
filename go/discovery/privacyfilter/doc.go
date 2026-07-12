// Package privacyfilter provides content filtering before indexing.
//
// Domain:  Discovery
// Concern: What content should NOT enter the index?
//
// Pre-index privacy gate (Kissner panel finding). Skips sensitive files
// (.env, credentials, keys) and redacts sensitive patterns (tokens, passwords).
// Runs before indexing, not after triage.
package privacyfilter
