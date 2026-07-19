// Package cloudrun provisions Cloud Run services and jobs in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we create Cloud Run services and jobs in GCP?
//
// Services are long-running HTTP handlers. Jobs are batch tasks.
// Both use IgnoreChanges on container image — deploys happen via CI/CD.
package cloudrun
