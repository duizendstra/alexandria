// Package dataform provisions Dataform repositories, release configs,
// and workflow configs in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we provision Dataform resources in GCP?
//
// This adapter creates repositories connected to Git remotes with
// auth tokens, and optionally configures release and workflow schedules.
package dataform
