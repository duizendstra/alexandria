// Package tagkeys provisions classification dimensions as tag keys in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we create classification dimensions (tag keys) on the GCP platform?
//
// This platform adapter implements the classification domain using Pulumi
// and the GCP Tags API. It validates the definitions via the domain package,
// then creates the resources with Pulumi protection enabled.
package tagkeys
