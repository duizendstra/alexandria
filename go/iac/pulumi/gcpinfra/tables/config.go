package tables

import "errors"

var (
	// ErrNameRequired means the table has no identifier.
	ErrNameRequired = errors.New("tables: Name is required")
	// ErrSchemaRequired means the table has no schema.
	ErrSchemaRequired = errors.New("tables: Schema is required")
	// ErrExternalNameRequired means the external table has no identifier.
	ErrExternalNameRequired = errors.New("tables: external Name is required")
	// ErrSourceFormatRequired means the external table has no data format.
	ErrSourceFormatRequired = errors.New("tables: external SourceFormat is required")
	// ErrSourceURIsRequired means the external table has no source URIs.
	ErrSourceURIsRequired = errors.New("tables: external SourceURIs is required")
)

// Config defines a BigQuery table to be provisioned.
type Config struct {
	// Name is the Pulumi resource name and table ID.
	Name string
	// Schema is the JSON-encoded table schema.
	Schema string
	// PartitionField is the column to partition by (time-based, DAY granularity).
	// Leave empty for unpartitioned tables.
	PartitionField string
	// DeletionProtection prevents accidental deletion.
	DeletionProtection bool
	// Labels are resource labels.
	Labels map[string]string
}

// Validate checks that the table configuration is complete.
func (c *Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Schema == "" {
		return ErrSchemaRequired
	}

	return nil
}

// ExternalConfig defines a BigQuery external table (e.g. Google Sheets).
type ExternalConfig struct {
	// Name is the Pulumi resource name and table ID.
	Name string
	// Schema is the JSON-encoded table schema.
	Schema string
	// SourceFormat is the external data format (e.g. "GOOGLE_SHEETS").
	SourceFormat string
	// SourceURIs are the source URIs.
	SourceURIs []string
	// SheetRange is the sheet range (Sheets only).
	SheetRange string
	// SkipLeadingRows skips header rows.
	SkipLeadingRows int
	// DeletionProtection prevents accidental deletion.
	DeletionProtection bool
	// Labels are resource labels.
	Labels map[string]string
}

// Validate checks that the external table configuration is complete.
func (c *ExternalConfig) Validate() error {
	if c.Name == "" {
		return ErrExternalNameRequired
	}
	if c.SourceFormat == "" {
		return ErrSourceFormatRequired
	}
	if len(c.SourceURIs) == 0 {
		return ErrSourceURIsRequired
	}

	return nil
}
