// Package classification defines the domain model for resource classification vocabulary.
//
// BC:      Governance
// Concern: What is a valid resource classification dimension?
//
// This package is pure Go — zero external dependencies. It defines the
// Dimension type that describes a classification axis and validates
// it before any provisioning tool creates the actual resources.
//
// Cloud-agnostic — GCP calls these "tag keys", AWS and Azure call
// them "tags". The concept is universal: a named dimension with a
// description that classifies resources for cost, ownership, or compliance.
//
// Adapters implement the provisioning using platform-specific SDKs.
package classification
