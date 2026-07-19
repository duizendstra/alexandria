package tables_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tables"
)

func TestValidateValid(t *testing.T) {
	c := tables.Config{Name: tableEvents, Schema: `[{"name":"id","type":"STRING","mode":"REQUIRED"}]`}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	c := tables.Config{Schema: "s"}
	if err := c.Validate(); !errors.Is(err, tables.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
}

func TestValidateMissingSchema(t *testing.T) {
	c := tables.Config{Name: "n"}
	if err := c.Validate(); !errors.Is(err, tables.ErrSchemaRequired) {
		t.Errorf("expected ErrSchemaRequired, got %v", err)
	}
}

func TestExternalValidateValid(t *testing.T) {
	c := tables.ExternalConfig{
		Name:         tableForecast,
		SourceFormat: formatSheets,
		SourceURIs:   []string{"https://docs.google.com/spreadsheets/d/example"},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExternalValidateMissingName(t *testing.T) {
	c := tables.ExternalConfig{SourceFormat: "f", SourceURIs: []string{"u"}}
	if err := c.Validate(); !errors.Is(err, tables.ErrExternalNameRequired) {
		t.Errorf("expected ErrExternalNameRequired, got %v", err)
	}
}

func TestExternalValidateMissingSourceFormat(t *testing.T) {
	c := tables.ExternalConfig{Name: "n", SourceURIs: []string{"u"}}
	if err := c.Validate(); !errors.Is(err, tables.ErrSourceFormatRequired) {
		t.Errorf("expected ErrSourceFormatRequired, got %v", err)
	}
}

func TestExternalValidateMissingSourceURIs(t *testing.T) {
	c := tables.ExternalConfig{Name: "n", SourceFormat: "f"}
	if err := c.Validate(); !errors.Is(err, tables.ErrSourceURIsRequired) {
		t.Errorf("expected ErrSourceURIsRequired, got %v", err)
	}
}
