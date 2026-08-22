// Package filelock implements file-based mutual exclusion satisfying coordination.Excluder.
//
// Layer:   Platform
// Concern: Prevent concurrent executions on the same host from running on the same subject
// using atomic filesystem lock files with automatic signal recovery.
package filelock
