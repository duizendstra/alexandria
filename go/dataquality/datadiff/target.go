package datadiff

// Target identifies a schema-based database table relation to compare.
// It uses database-agnostic abstractions while maintaining backward-compatible
// properties for GCP BigQuery targets.
type Target struct {
	Project string // Database or GCP project qualifier (e.g. "my-project")
	Dataset string // Dataset or schema context (e.g. "my_dataset")
	Table   string // Table or relation identifier (e.g. "users")
}

// Qualifier returns the database or project qualifier context.
func (t Target) Qualifier() string { return t.Project }

// Schema returns the dataset or schema context.
func (t Target) Schema() string { return t.Dataset }

// Relation returns the table or relation identifier.
func (t Target) Relation() string { return t.Table }

// String returns a fully qualified database relation reference.
func (t Target) String() string {
	return t.Project + "." + t.Dataset + "." + t.Table
}

// FullyQualified returns the backtick-quoted reference (suited for SQL systems like BigQuery).
func (t Target) FullyQualified() string {
	return "`" + t.Project + "." + t.Dataset + "." + t.Table + "`"
}

// ColumnMode describes the nullability and cardinality of a column.
type ColumnMode string

const (
	ModeNullable ColumnMode = "NULLABLE"
	ModeRequired ColumnMode = "REQUIRED"
	ModeRepeated ColumnMode = "REPEATED" // array
)

// Column describes a table column, including nested structures.
// For RECORD/STRUCT types, Children contains the sub-fields.
// For REPEATED RECORD, Mode is REPEATED and Children is populated.
type Column struct {
	Name     string
	DataType string     // e.g. "STRING", "INT64", "RECORD", "FLOAT64"
	Mode     ColumnMode // NULLABLE, REQUIRED, or REPEATED
	Position int
	Children []Column // populated when DataType is "RECORD"
}

// IsNested returns true if this column has child fields.
func (c *Column) IsNested() bool {
	return len(c.Children) > 0
}

// IsRepeated returns true if this column is an array.
func (c *Column) IsRepeated() bool {
	return c.Mode == ModeRepeated
}

// FlattenTo recursively appends all leaf columns with dot-notation paths
// to the pre-allocated accumulator slice, avoiding heap allocations of intermediate slices.
func (c *Column) FlattenTo(prefix string, acc []Column) []Column {
	path := c.Name
	if prefix != "" {
		path = prefix + "." + c.Name
	}

	if !c.IsNested() {
		return append(acc, Column{
			Name:     path,
			DataType: c.DataType,
			Mode:     c.Mode,
			Position: c.Position,
		})
	}

	for _, child := range c.Children {
		acc = child.FlattenTo(path, acc)
	}

	return acc
}

// Flatten returns all leaf columns with dot-notation paths.
func (c *Column) Flatten(prefix string) []Column {
	return c.FlattenTo(prefix, nil)
}

// FlattenAll returns all leaf columns from a slice, expanding nested fields.
// Performs in-place pre-allocation accumulation to maximize performance.
func FlattenAll(cols []Column) []Column {
	var flat []Column
	for _, c := range cols {
		flat = c.FlattenTo("", flat)
	}

	return flat
}

// TableMeta describes a table's structure and partitioning.
type TableMeta struct {
	Columns         []Column
	RowCount        int64
	PartitionColumn string // empty if not partitioned
	ClusterColumns  []string
	SizeBytes       int64
}
