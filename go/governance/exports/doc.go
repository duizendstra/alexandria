// Package exports defines the governance stack output names.
//
// BC:      Governance
// Concern: What outputs does the governance stack export?
//
// These constants are the contract between the governance producer
// and downstream consumers (identity, delivery, environment, etc.).
// Changing a value here is a breaking change for all consumers.
//
// Consumers import this package to reference exports by constant
// instead of string literal — enabling compile-time verification.
package exports
