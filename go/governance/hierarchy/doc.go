// Package hierarchy defines the domain model for organizational container trees.
//
// BC:      Governance
// Concern: What is a valid organizational hierarchy?
//
// This package is pure Go — zero external dependencies. It defines the
// Config type that describes a desired container structure and validates
// structural correctness before any provisioning tool creates resources.
//
// Cloud-agnostic — GCP calls these "folders", AWS calls them "OUs",
// Azure calls them "management groups". This package models the
// universal concept: a root container with named children.
//
// Cloud-specific parent format validation belongs in the adapter,
// not here. This package validates structure only.
package hierarchy
