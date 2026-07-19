// Package folders provisions organizational container hierarchies in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we create folder hierarchies on the GCP platform?
//
// This platform adapter implements the hierarchy domain using Pulumi
// and the GCP Organizations API. It validates the GCP-specific parent
// format (organizations/ID or folders/ID), then creates the resources
// with deletion protection and Pulumi protection enabled.
//
// The GCP parent format validation lives here — not in the domain —
// because it is cloud-specific. Other platform adapters (AWS, Azure)
// would validate their own parent formats.
package folders
