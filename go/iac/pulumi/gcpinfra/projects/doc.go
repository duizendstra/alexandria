// Package projects provisions GCP projects with API enablement.
//
// Layer:   IaC Building Block
// Concern: How do we create a project and enable APIs in GCP?
//
// This adapter creates a project with deletion protection, disables
// auto-created default VPC, and enables the requested APIs sequentially
// (APIs depend on project existence).
package projects
