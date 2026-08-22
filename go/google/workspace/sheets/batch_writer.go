package sheets

import (
	"fmt"
	"strings"
)

const (
	alphabetSize           = 26
	InputOptionRaw         = "RAW"
	InputOptionUserEntered = "USER_ENTERED"
)

// valueUpdateBatch represents a single range update with specific input parsing semantics.
type valueUpdateBatch struct {
	Range            string
	ValueInputOption string
	Values           [][]any
}

// ColIndexToA1 converts a 0-based column index to spreadsheet column letters (e.g. 0 -> "A", 25 -> "Z", 26 -> "AA").
func ColIndexToA1(colIdx int) string {
	if colIdx < 0 {
		return "A"
	}
	res := ""
	for {
		rem := colIdx % alphabetSize
		res = string(rune('A'+rem)) + res
		colIdx = colIdx/alphabetSize - 1
		if colIdx < 0 {
			break
		}
	}

	return res
}

// EscapeSheetTitle returns a safely quoted sheet title for A1 notation.
func EscapeSheetTitle(title string) string {
	return "'" + strings.ReplaceAll(title, "'", "''") + "'"
}

// prepareValueUpdates partitions a Table into safe, batched value updates.
// It guarantees that:
// 1. If no cells contain formulas, the entire table (headers + data) is emitted as a single RAW update.
// 2. If formulas exist, headers and non-formula columns are emitted as RAW, while formula columns
// are emitted as USER_ENTERED. A column holding both kinds of cell is split into runs of rows
// so that only its formula cells go USER_ENTERED (#244). Untrusted strings cannot trigger
// formula execution.
func prepareValueUpdates(tabTitle string, table *Table) []valueUpdateBatch {
	if table == nil || (len(table.Headers) == 0 && len(table.Rows) == 0) {
		return nil
	}

	escapedTitle := EscapeSheetTitle(tabTitle)
	numCols := table.ColCount()
	if numCols == 0 {
		return nil
	}

	hasFormulas := checkTableHasFormulas(table)

	// Fast path: No formulas in any cell. Write everything as a single RAW update.
	if !hasFormulas {
		return buildAllRawBatch(escapedTitle, table)
	}

	// Mixed mode: Write headers first as RAW (row 1), then partition data columns (row 2+).
	return buildMixedModeBatches(escapedTitle, table, numCols)
}

func checkTableHasFormulas(table *Table) bool {
	for _, r := range table.Rows {
		for _, c := range r {
			if c.IsFormula {
				return true
			}
		}
	}

	return false
}

func buildAllRawBatch(escapedTitle string, table *Table) []valueUpdateBatch {
	var allRows [][]any
	if len(table.Headers) > 0 {
		headerRow := make([]any, len(table.Headers))
		for i, h := range table.Headers {
			headerRow[i] = h
		}
		allRows = append(allRows, headerRow)
	}
	for _, r := range table.Rows {
		row := make([]any, len(r))
		for i, c := range r {
			row[i] = c.RawVal
		}
		allRows = append(allRows, row)
	}

	return []valueUpdateBatch{
		{
			Range:            escapedTitle + "!A1",
			ValueInputOption: InputOptionRaw,
			Values:           allRows,
		},
	}
}

// colKind classifies how a data column is written.
type colKind uint8

const (
	// colRaw has no formula cells: the column is written RAW as one block.
	colRaw colKind = iota
	// colFormula has nothing but formula cells: the column is written USER_ENTERED as one block.
	colFormula
	// colMixed holds both formula and non-formula cells: it is written per run of rows, so the
	// value-input option is chosen per cell and text cells keep their formula-injection immunity.
	colMixed
)

type colSlice struct {
	startCol int
	endCol   int
	kind     colKind
}

func buildMixedModeBatches(escapedTitle string, table *Table, numCols int) []valueUpdateBatch {
	var batches []valueUpdateBatch
	startDataRow := 1

	if len(table.Headers) > 0 {
		headerRow := make([]any, len(table.Headers))
		for i, h := range table.Headers {
			headerRow[i] = h
		}
		batches = append(batches, valueUpdateBatch{
			Range:            fmt.Sprintf("%s!A1:%s1", escapedTitle, ColIndexToA1(len(table.Headers)-1)),
			ValueInputOption: InputOptionRaw,
			Values:           [][]any{headerRow},
		})
		startDataRow = 2
	}

	if len(table.Rows) == 0 {
		return batches
	}

	slices := partitionColumns(table, numCols)
	endDataRow := startDataRow + len(table.Rows) - 1

	for _, s := range slices {
		if s.kind == colMixed {
			batches = append(batches, buildMixedColumnBatches(escapedTitle, table, s.startCol, startDataRow)...)

			continue
		}

		sliceValues := extractSliceValues(table, s)
		opt := InputOptionRaw
		if s.kind == colFormula {
			opt = InputOptionUserEntered
		}

		startA1 := fmt.Sprintf("%s%d", ColIndexToA1(s.startCol), startDataRow)
		endA1 := fmt.Sprintf("%s%d", ColIndexToA1(s.endCol), endDataRow)

		batches = append(batches, valueUpdateBatch{
			Range:            fmt.Sprintf("%s!%s:%s", escapedTitle, startA1, endA1),
			ValueInputOption: opt,
			Values:           sliceValues,
		})
	}

	return batches
}

// buildMixedColumnBatches splits one mixed column into runs of contiguous rows that share
// the same value-input option: formula cells go USER_ENTERED, everything else — including
// cells a short row leaves missing — goes RAW, exactly as a pure text column would.
func buildMixedColumnBatches(escapedTitle string, table *Table, colIdx, startDataRow int) []valueUpdateBatch {
	var batches []valueUpdateBatch
	colA1 := ColIndexToA1(colIdx)

	flush := func(runStart, runEnd int, isFormula bool) {
		values := make([][]any, 0, runEnd-runStart+1)
		for r := runStart; r <= runEnd; r++ {
			values = append(values, []any{cellValueAt(table.Rows[r], colIdx)})
		}
		opt := InputOptionRaw
		if isFormula {
			opt = InputOptionUserEntered
		}
		batches = append(batches, valueUpdateBatch{
			Range:            fmt.Sprintf("%s!%s%d:%s%d", escapedTitle, colA1, startDataRow+runStart, colA1, startDataRow+runEnd),
			ValueInputOption: opt,
			Values:           values,
		})
	}

	runStart := 0
	runFormula := cellIsFormulaAt(table.Rows[0], colIdx)
	for r := 1; r < len(table.Rows); r++ {
		if f := cellIsFormulaAt(table.Rows[r], colIdx); f != runFormula {
			flush(runStart, r-1, runFormula)
			runStart, runFormula = r, f
		}
	}
	flush(runStart, len(table.Rows)-1, runFormula)

	return batches
}

// cellIsFormulaAt reports whether row has a formula cell at colIdx; a missing cell is not a formula.
func cellIsFormulaAt(row []Cell, colIdx int) bool {
	return colIdx < len(row) && row[colIdx].IsFormula
}

// cellValueAt returns the raw value at colIdx, or "" when the row is too short.
func cellValueAt(row []Cell, colIdx int) any {
	if colIdx < len(row) {
		return row[colIdx].RawVal
	}

	return ""
}

// partitionColumns classifies every data column and groups adjacent columns of the same
// kind into one slice. A mixed column never merges with a neighbour: it is written per cell.
func partitionColumns(table *Table, numCols int) []colSlice {
	kinds := classifyColumns(table, numCols)

	var slices []colSlice
	currStart := 0

	for c := 1; c < numCols; c++ {
		if kinds[c] != kinds[currStart] || kinds[c] == colMixed {
			slices = append(slices, colSlice{startCol: currStart, endCol: c - 1, kind: kinds[currStart]})
			currStart = c
		}
	}
	slices = append(slices, colSlice{startCol: currStart, endCol: numCols - 1, kind: kinds[currStart]})

	return slices
}

// classifyColumns decides each column's kind from the cells that are present; a cell a short
// row leaves missing counts for neither side, so such gaps are padded as they always were.
func classifyColumns(table *Table, numCols int) []colKind {
	formulaCells := make([]int, numCols)
	otherCells := make([]int, numCols)
	for _, r := range table.Rows {
		for colIdx := 0; colIdx < len(r) && colIdx < numCols; colIdx++ {
			if r[colIdx].IsFormula {
				formulaCells[colIdx]++
			} else {
				otherCells[colIdx]++
			}
		}
	}

	kinds := make([]colKind, numCols)
	for c := range numCols {
		switch {
		case formulaCells[c] == 0:
			kinds[c] = colRaw
		case otherCells[c] == 0:
			kinds[c] = colFormula
		default:
			kinds[c] = colMixed
		}
	}

	return kinds
}

func extractSliceValues(table *Table, s colSlice) [][]any {
	sliceValues := make([][]any, len(table.Rows))
	width := s.endCol - s.startCol + 1

	for rIdx, r := range table.Rows {
		rowVals := make([]any, width)
		for offset := range width {
			rowVals[offset] = cellValueAt(r, s.startCol+offset)
		}
		sliceValues[rIdx] = rowVals
	}

	return sliceValues
}
