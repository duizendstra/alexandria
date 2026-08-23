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
	// ErrDuplicateName means two tables in one Apply call share a Name. The
	// name is the Pulumi logical name, so a repeat would collide URNs.
	ErrDuplicateName = errors.New("tables: duplicate table name")
	// ErrDuplicateExternalName means two external tables in one
	// ApplyExternal call share a Name.
	ErrDuplicateExternalName = errors.New("tables: duplicate external table name")
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
	// DeletionProtection forces GCP-level deletion protection on even when
	// the stack is ephemeral. A permanent stack protects every table anyway,
	// so this only matters to a stack that opted out with lifecycle.Ephemeral.
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
	// DeletionProtection forces GCP-level deletion protection on even when
	// the stack is ephemeral. A permanent stack protects every table anyway,
	// so this only matters to a stack that opted out with lifecycle.Ephemeral.
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
