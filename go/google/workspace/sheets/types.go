package sheets

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/api/sheets/v4"
)

const (
	colorMax          = 255.0
	hexLen            = 6
	DefaultSheetTitle = "Sheet1"
)

var (
	// ErrNilType indicates that a nil type was provided to FromStructs.
	ErrNilType = errors.New("sheets: nil type passed to FromStructs")

	// ErrInvalidStructSlice indicates that FromStructs received a non-struct slice.
	ErrInvalidStructSlice = errors.New("sheets: FromStructs expects a slice of structs or struct pointers")
)

// Cell represents a single table cell with explicit value injection protection semantics.
type Cell struct {
	// RawVal is the underlying Go value (string, int64, float64, bool, etc.).
	RawVal any
	// IsFormula indicates whether this cell contains an executable spreadsheet formula.
	// When true, the cell will be written using ValueInputOption("USER_ENTERED").
	// When false (the default), the cell will be written safely using ValueInputOption("RAW"),
	// ensuring any leading '=', '+', '-', or '@' characters are treated as literal text.
	IsFormula bool
}

// Text constructs a literal text cell. It is immune to formula injection.
func Text(v any) Cell {
	if v == nil {
		return Cell{RawVal: "", IsFormula: false}
	}

	return Cell{RawVal: fmt.Sprint(v), IsFormula: false}
}

// Formula constructs an explicit spreadsheet formula cell.
// It will be evaluated by Google Sheets upon insertion.
func Formula(f string) Cell {
	return Cell{RawVal: f, IsFormula: true}
}

// Hyperlink constructs an explicit =HYPERLINK("url"; "label") formula cell.
func Hyperlink(url, label string) Cell {
	if label == "" {
		label = url
	}
	// Escape quotes in url and label for sheets formula.
	safeURL := escapeFormulaString(url)
	safeLabel := escapeFormulaString(label)

	return Formula(`=HYPERLINK("` + safeURL + `";"` + safeLabel + `")`)
}

// escapeFormulaString escapes internal double quotes for formula string literals.
func escapeFormulaString(s string) string {
	var b []byte
	for i := range s {
		if s[i] == '"' {
			b = append(b, '"', '"')
		} else {
			b = append(b, s[i])
		}
	}

	return string(b)
}

// Number constructs a numeric cell with safe raw representation.
func Number[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](v T) Cell {
	return Cell{RawVal: v, IsFormula: false}
}

// Bool constructs a boolean cell.
func Bool(b bool) Cell {
	return Cell{RawVal: b, IsFormula: false}
}

// Time constructs a formatted date/time cell.
func Time(t time.Time, layout ...string) Cell {
	l := time.RFC3339
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}

	return Cell{RawVal: t.Format(l), IsFormula: false}
}

// Table represents a two-dimensional structured dataset with headers and typed cells.
type Table struct {
	// Headers is the top header row.
	Headers []string
	// Rows contains the data rows.
	Rows [][]Cell
	// ColumnWidths specifies custom pixel widths per column index (0-based).
	ColumnWidths map[int]int64
}

// Color represents an RGBA color with normalized 0.0 - 1.0 float values.
type Color struct {
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

// ToSheetsColor converts Color to the Google Sheets API Color representation.
func (c Color) ToSheetsColor() *sheets.Color {
	return &sheets.Color{
		Red:   c.Red,
		Green: c.Green,
		Blue:  c.Blue,
		Alpha: c.Alpha,
	}
}

// RGB creates a Color from standard 0-255 RGB byte values.
func RGB(r, g, b uint8) Color {
	return Color{
		Red:   float64(r) / colorMax,
		Green: float64(g) / colorMax,
		Blue:  float64(b) / colorMax,
		Alpha: 1.0,
	}
}

// RGBA creates a Color from 0-255 RGB byte values and 0.0-1.0 alpha.
func RGBA(r, g, b uint8, alpha float64) Color {
	return Color{
		Red:   float64(r) / colorMax,
		Green: float64(g) / colorMax,
		Blue:  float64(b) / colorMax,
		Alpha: alpha,
	}
}

// Hex creates a Color from a 6-character hex string (e.g. "#1C4E78" or "1C4E78").
func Hex(h string) Color {
	if h != "" && h[0] == '#' {
		h = h[1:]
	}
	if len(h) != hexLen {
		return Color{Alpha: 1.0}
	}
	var r, g, b uint8
	_, _ = fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b)

	return RGB(r, g, b)
}

// Theme defines the declarative styling rules for a sheet tab.
type Theme struct {
	// HeaderBackground is the background fill color for header rows.
	HeaderBackground Color
	// HeaderForeground is the text color for header rows.
	HeaderForeground Color
	// HeaderBold sets bold text formatting for header rows.
	HeaderBold bool
	// EnableBanding enables alternating row zebra striping on the data rows.
	EnableBanding bool
	// ZebraFirstBand is the primary row background color.
	ZebraFirstBand Color
	// ZebraSecondBand is the alternating row background color.
	ZebraSecondBand Color
	// AutoFitColumns enables auto-resizing column widths to content width.
	AutoFitColumns bool
	// ColumnWidths specifies default pixel widths for specific column indices.
	ColumnWidths map[int]int64
}

// TabSpec defines the specification for synchronizing or creating a spreadsheet tab.
type TabSpec struct {
	// Title is the name of the tab.
	Title string
	// GID is the sheet tab ID. If >= 0, the syncer matches by GID first.
	GID int64
	// CreateIfMissing determines whether to create the tab if it does not exist.
	CreateIfMissing bool
	// FrozenRows is the number of top rows to freeze (defaults to 1 if Data has headers).
	FrozenRows int64
	// FrozenCols is the number of left columns to freeze (defaults to 0).
	FrozenCols int64
	// Theme specifies visual styling. If nil, default corporate navy theme is used.
	Theme *Theme
	// Data is the tabular data to write.
	Data *Table
}

// DocumentSpec defines the specification for creating or updating an entire spreadsheet.
type DocumentSpec struct {
	// Title is the spreadsheet title.
	Title string
	// Locale is the spreadsheet locale (e.g. "nl_NL", "en_US"). Defaults to "nl_NL".
	Locale string
	// FolderID is the optional Google Drive folder ID to move the newly created spreadsheet into.
	FolderID string
	// Tabs is the list of tabs to create or update.
	Tabs []TabSpec
	// PruneTabs, when true, deletes existing tabs in the spreadsheet that are not in Tabs.
	PruneTabs bool
}

// TabResult contains the result of a single tab synchronization.
type TabResult struct {
	// SheetID is the numeric GID of the tab.
	SheetID int64
	// Title is the title of the tab.
	Title string
	// RowCount is the total number of grid rows after update.
	RowCount int64
	// ColumnCount is the total number of grid columns after update.
	ColumnCount int64
	// RowsWritten is the number of data rows written.
	RowsWritten int
}

// DocumentResult contains the result of a full spreadsheet creation or update.
type DocumentResult struct {
	// SpreadsheetID is the Google Sheets spreadsheet identifier.
	SpreadsheetID string
	// SpreadsheetURL is the direct browser URL to open the spreadsheet.
	SpreadsheetURL string
	// Title is the title of the spreadsheet.
	Title string
	// Tabs contains results for each tab processed.
	Tabs []TabResult
}
