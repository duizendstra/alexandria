package sheets

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// NewTable initializes an empty Table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		Headers:           headers,
		Rows:              make([][]Cell, 0),
		ColumnWidths:      make(map[int]int64),
		ColumnConstraints: make(map[int]ColumnConstraint),
	}
}

// AddRow adds a single row of typed Cells to the table.
func (t *Table) AddRow(cells ...Cell) *Table {
	t.Rows = append(t.Rows, cells)

	return t
}

// AddRowValues converts raw Go values into safe literal cells (Text, Number, Bool, Time)
// and appends them as a new row. Values starting with '=' are treated as raw text and
// will NOT be interpreted as formulas.
func (t *Table) AddRowValues(vals ...any) *Table {
	cells := make([]Cell, len(vals))
	for i, v := range vals {
		cells[i] = valueToCell(v, false)
	}
	t.Rows = append(t.Rows, cells)

	return t
}

// SetColumnWidth configures a fixed pixel width for a specific column index (0-based).
func (t *Table) SetColumnWidth(colIdx int, pixelSize int64) *Table {
	if t.ColumnWidths == nil {
		t.ColumnWidths = make(map[int]int64)
	}
	if t.ColumnConstraints == nil {
		t.ColumnConstraints = make(map[int]ColumnConstraint)
	}
	t.ColumnWidths[colIdx] = pixelSize
	c := t.ColumnConstraints[colIdx]
	c.Width = pixelSize
	t.ColumnConstraints[colIdx] = c

	return t
}

// SetColumnWidthByName configures a fixed pixel width for the column matching the header name (case-insensitive).
func (t *Table) SetColumnWidthByName(header string, pixelSize int64) *Table {
	for i, h := range t.Headers {
		if strings.EqualFold(h, header) {
			return t.SetColumnWidth(i, pixelSize)
		}
	}

	return t
}

// SetColumnBounds configures minimum and maximum pixel bounds for a specific column index (0-based).
// If minWidth > 0, it sets or updates the minimum bound. If maxWidth > 0, it sets or updates the maximum bound.
func (t *Table) SetColumnBounds(colIdx int, minWidth, maxWidth int64) *Table {
	if t.ColumnConstraints == nil {
		t.ColumnConstraints = make(map[int]ColumnConstraint)
	}
	c := t.ColumnConstraints[colIdx]
	if minWidth > 0 {
		c.MinWidth = minWidth
	}
	if maxWidth > 0 {
		c.MaxWidth = maxWidth
	}
	t.ColumnConstraints[colIdx] = c

	return t
}

// SetColumnBoundsByName configures minimum and maximum pixel bounds for the column matching the header name (case-insensitive).
func (t *Table) SetColumnBoundsByName(header string, minWidth, maxWidth int64) *Table {
	for i, h := range t.Headers {
		if strings.EqualFold(h, header) {
			return t.SetColumnBounds(i, minWidth, maxWidth)
		}
	}

	return t
}

// SetColumnMaxWidthByName configures a maximum pixel bound for the column matching the header name (case-insensitive).
func (t *Table) SetColumnMaxWidthByName(header string, maxWidth int64) *Table {
	return t.SetColumnBoundsByName(header, 0, maxWidth)
}

// SetColumnMinWidthByName configures a minimum pixel bound for the column matching the header name (case-insensitive).
func (t *Table) SetColumnMinWidthByName(header string, minWidth int64) *Table {
	return t.SetColumnBoundsByName(header, minWidth, 0)
}

// RowCount returns the number of data rows (excluding headers).
func (t *Table) RowCount() int {
	return len(t.Rows)
}

// ColCount returns the maximum number of columns across headers and rows.
func (t *Table) ColCount() int {
	maxCols := len(t.Headers)
	for _, r := range t.Rows {
		if len(r) > maxCols {
			maxCols = len(r)
		}
	}

	return maxCols
}

// structFieldSpec captures metadata parsed from a struct field.
type structFieldSpec struct {
	fieldIndex int
	header     string
	width      int64
	minWidth   int64
	maxWidth   int64
	isFormula  bool
}

// FromStructs uses reflection to convert a slice of structs or struct pointers into a Table.
// Struct fields can be annotated with the `sheets` tag:
//
//	type Employee struct {
//	    ID       string `sheets:"Employee ID,width=120"`
//	    Name     string `sheets:"Full Name,maxWidth=250"`
//	    Active   bool   `sheets:"Is Active"`
//	    Profile  string `sheets:"Profile Link,formula"` // evaluated as formula
//	    Internal string `sheets:"-"`                    // omitted from sheet
//	}
func FromStructs[T any](items []T) (*Table, error) {
	var zero T
	tType := reflect.TypeOf(zero)
	if tType == nil {
		return nil, ErrNilType
	}
	if tType.Kind() == reflect.Pointer {
		tType = tType.Elem()
	}
	if tType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: got %s", ErrInvalidStructSlice, tType.Kind())
	}

	specs := parseStructFields(tType)
	headers := make([]string, len(specs))
	widths := make(map[int]int64)
	constraints := make(map[int]ColumnConstraint)

	for i, spec := range specs {
		headers[i] = spec.header
		c := ColumnConstraint{
			Width:    spec.width,
			MinWidth: spec.minWidth,
			MaxWidth: spec.maxWidth,
		}
		if c.Width > 0 || c.MinWidth > 0 || c.MaxWidth > 0 {
			constraints[i] = c
		}
		if spec.width > 0 {
			widths[i] = spec.width
		}
	}

	tbl := &Table{
		Headers:           headers,
		Rows:              make([][]Cell, 0, len(items)),
		ColumnWidths:      widths,
		ColumnConstraints: constraints,
	}

	for _, item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Pointer {
			if val.IsNil() {
				continue
			}
			val = val.Elem()
		}

		row := make([]Cell, len(specs))
		for colIdx, spec := range specs {
			fVal := val.Field(spec.fieldIndex)
			row[colIdx] = fieldValueToCell(fVal, spec.isFormula)
		}
		tbl.Rows = append(tbl.Rows, row)
	}

	return tbl, nil
}

func parseStructFields(t reflect.Type) []structFieldSpec {
	var specs []structFieldSpec
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("sheets")
		if tag == "-" {
			continue
		}

		spec := parseFieldTag(i, field.Name, tag)
		specs = append(specs, spec)
	}

	return specs
}

func parseFieldTag(fieldIndex int, defaultName, tag string) structFieldSpec {
	spec := structFieldSpec{
		fieldIndex: fieldIndex,
		header:     defaultName,
	}
	if tag == "" {
		return spec
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		spec.header = strings.TrimSpace(parts[0])
	}
	for _, opt := range parts[1:] {
		parseTagOption(&spec, strings.TrimSpace(opt))
	}

	return spec
}

func parseTagOption(spec *structFieldSpec, opt string) {
	if opt == "formula" {
		spec.isFormula = true

		return
	}

	if rest, ok := cutAnyPrefix(opt, "width="); ok {
		if w, err := strconv.ParseInt(rest, 10, 64); err == nil {
			spec.width = w
		}

		return
	}

	if rest, ok := cutAnyPrefix(opt, "maxWidth=", "max="); ok {
		if w, err := strconv.ParseInt(rest, 10, 64); err == nil {
			spec.maxWidth = w
		}

		return
	}

	if rest, ok := cutAnyPrefix(opt, "minWidth=", "min="); ok {
		if w, err := strconv.ParseInt(rest, 10, 64); err == nil {
			spec.minWidth = w
		}
	}
}

func cutAnyPrefix(s string, prefixes ...string) (string, bool) {
	for _, p := range prefixes {
		if rest, ok := strings.CutPrefix(s, p); ok {
			return rest, true
		}
	}

	return "", false
}

func fieldValueToCell(v reflect.Value, forceFormula bool) Cell {
	if !v.IsValid() {
		return Cell{RawVal: "", IsFormula: false}
	}

	// Direct Cell type support.
	if v.Type() == reflect.TypeFor[Cell]() {
		if c, ok := v.Interface().(Cell); ok {
			return c
		}
	}

	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return Cell{RawVal: "", IsFormula: false}
		}
		v = v.Elem()
	}

	return valueToCell(v.Interface(), forceFormula)
}

func valueToCell(val any, forceFormula bool) Cell {
	if val == nil {
		return Cell{RawVal: "", IsFormula: false}
	}

	if c, ok := val.(Cell); ok {
		return c
	}

	if forceFormula {
		return Formula(fmt.Sprint(val))
	}

	if numCell, ok := numericToCell(val); ok {
		return numCell
	}

	switch v := val.(type) {
	case string:
		return Text(v)
	case bool:
		return Bool(v)
	case time.Time:
		return Time(v)
	case fmt.Stringer:
		return Text(v.String())
	default:
		return Text(fmt.Sprint(v))
	}
}

func numericToCell(val any) (Cell, bool) {
	switch v := val.(type) {
	case int:
		return Number(int64(v)), true
	case int8:
		return Number(int64(v)), true
	case int16:
		return Number(int64(v)), true
	case int32:
		return Number(int64(v)), true
	case int64:
		return Number(v), true
	case uint:
		return Number(uint64(v)), true
	case uint8:
		return Number(uint64(v)), true
	case uint16:
		return Number(uint64(v)), true
	case uint32:
		return Number(uint64(v)), true
	case uint64:
		return Number(v), true
	case float32:
		return Number(float64(v)), true
	case float64:
		return Number(v), true
	default:
		return Cell{}, false
	}
}
