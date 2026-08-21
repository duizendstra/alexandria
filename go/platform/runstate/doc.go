// Package runstate keeps the on-disk state of a run: an exclusive lock per
// subject, and a short-lived lease that proves a check was passed.
//
// Layer:   Platform
// Concern: Let a command refuse to run twice at once, and refuse to run at all
// unless a recent check still applies to exactly this subject and this build.
package runstate
