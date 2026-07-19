package firestore_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/firestore"
)

const (
	collectionConfig = "config"
	docConnectorA    = "connector-a"
)

func TestDatabaseValidateValid(t *testing.T) {
	c := firestore.DatabaseConfig{Name: "appdb", Region: "europe-west4"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDatabaseValidateMissingName(t *testing.T) {
	c := firestore.DatabaseConfig{Region: "r"}
	if err := c.Validate(); !errors.Is(err, firestore.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
}

func TestDatabaseValidateMissingRegion(t *testing.T) {
	c := firestore.DatabaseConfig{Name: "n"}
	if err := c.Validate(); !errors.Is(err, firestore.ErrRegionRequired) {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestDocumentValidateValid(t *testing.T) {
	c := firestore.DocumentConfig{Collection: collectionConfig, DocumentID: docConnectorA, Fields: `{"enabled":{"booleanValue":true}}`}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDocumentValidateMissingCollection(t *testing.T) {
	c := firestore.DocumentConfig{DocumentID: "d", Fields: "f"}
	if err := c.Validate(); !errors.Is(err, firestore.ErrCollectionRequired) {
		t.Errorf("expected ErrCollectionRequired, got %v", err)
	}
}

func TestDocumentValidateMissingDocumentID(t *testing.T) {
	c := firestore.DocumentConfig{Collection: "c", Fields: "f"}
	if err := c.Validate(); !errors.Is(err, firestore.ErrDocumentIDRequired) {
		t.Errorf("expected ErrDocumentIDRequired, got %v", err)
	}
}

func TestDocumentValidateMissingFields(t *testing.T) {
	c := firestore.DocumentConfig{Collection: "c", DocumentID: "d"}
	if err := c.Validate(); !errors.Is(err, firestore.ErrFieldsRequired) {
		t.Errorf("expected ErrFieldsRequired, got %v", err)
	}
}
