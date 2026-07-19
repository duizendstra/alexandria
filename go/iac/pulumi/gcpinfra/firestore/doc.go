// Package firestore provisions Firestore databases and documents in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we create Firestore databases and seed documents in GCP?
//
// Documents use IgnoreChanges on fields — config is seeded once,
// then managed via the application at runtime.
package firestore
