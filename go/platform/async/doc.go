// Package async provides an in-memory async task runner. Callers submit
// context-aware work functions that execute in goroutines. Each task gets
// a unique ID and progresses through pending → running → done/failed
// states. Results are retained in memory for polling until pruned —
// manually via Prune or automatically via the optional WithJanitor
// background reaper. Closing the runner cancels the contexts of all
// outstanding tasks.
//
// This is a cross-cutting infrastructure package — any bounded context
// can use it to make synchronous operations asynchronous (quality gate,
// build, deploy, infra provisioning).
//
// Domain:  Platform
// Concern: How do we execute long-running operations asynchronously?
package async
