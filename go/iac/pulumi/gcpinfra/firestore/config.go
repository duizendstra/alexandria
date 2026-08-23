package firestore

import (
	"errors"
	"fmt"
)

// The two delete-protection states the Firestore API accepts. Anything else,
// including the empty string, reads as protection disabled at the provider.
const (
	DeleteProtectionEnabled  = "DELETE_PROTECTION_ENABLED"
	DeleteProtectionDisabled = "DELETE_PROTECTION_DISABLED"
)

var (
	// ErrNameRequired means the database has no ID.
	ErrNameRequired = errors.New("firestore: Name is required")
	// ErrRegionRequired means the database has no location.
	ErrRegionRequired = errors.New("firestore: Region is required")
	// ErrInvalidDeleteProtection means DeleteProtection is neither empty nor
	// one of the two states the API accepts. An unrecognised value used to
	// reach the provider unchecked, where it read as protection disabled.
	ErrInvalidDeleteProtection = errors.New("firestore: DeleteProtection must be DELETE_PROTECTION_ENABLED or DELETE_PROTECTION_DISABLED")
	// ErrCollectionRequired means the document has no collection.
	ErrCollectionRequired = errors.New("firestore: document Collection is required")
	// ErrDocumentIDRequired means the document has no identifier.
	ErrDocumentIDRequired = errors.New("firestore: document DocumentID is required")
	// ErrFieldsRequired means the document has no field map.
	ErrFieldsRequired = errors.New("firestore: document Fields is required")
	// ErrDuplicateDocumentID means two documents in one ApplyDocuments call
	// share a DocumentID. The ID is the Pulumi logical name, so it must be
	// unique across collections within the call, not only within one.
	ErrDuplicateDocumentID = errors.New("firestore: duplicate document DocumentID")
)

// DatabaseConfig defines a Firestore database to be provisioned.
type DatabaseConfig struct {
	// Name is the database ID (e.g. "appdb").
	Name string
	// Region is the location (e.g. "europe-west4").
	Region string
	// DeleteProtection is "DELETE_PROTECTION_ENABLED" or
	// "DELETE_PROTECTION_DISABLED". Leave it empty to follow the stack: a
	// permanent stack enables it, lifecycle.Ephemeral disables it.
	DeleteProtection string
}

// Validate checks that the database configuration is complete.
func (c *DatabaseConfig) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Region == "" {
		return ErrRegionRequired
	}

	switch c.DeleteProtection {
	case "", DeleteProtectionEnabled, DeleteProtectionDisabled:
	default:
		return fmt.Errorf("%w, got %q", ErrInvalidDeleteProtection, c.DeleteProtection)
	}

	return nil
}

// DocumentConfig defines a Firestore document to be seeded.
type DocumentConfig struct {
	// Collection is the Firestore collection (e.g. "config").
	Collection string
	// DocumentID is the document identifier (e.g. "connector-a").
	DocumentID string
	// Fields is the JSON-encoded field map.
	Fields string
}

// Validate checks that the document configuration is complete.
func (c *DocumentConfig) Validate() error {
	if c.Collection == "" {
		return ErrCollectionRequired
	}
	if c.DocumentID == "" {
		return ErrDocumentIDRequired
	}
	if c.Fields == "" {
		return ErrFieldsRequired
	}

	return nil
}
