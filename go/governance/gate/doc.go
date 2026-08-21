// Package gate provides a pure Go policy verification gate and pre-flight
// decision engine for rollout plans, migration waves, and infrastructure changes.
//
// Gates evaluate a collection of named rules against configurable policies
// (Strict, Standard, Permissive) and produce auditable reports detailing
// pass, fail, anomaly, and skipped outcomes with evidence trails.
//
// Gate seamlessly interoperates with the invariant rule evaluation package
// via FromCheck and FromBuilder adapters.
package gate
