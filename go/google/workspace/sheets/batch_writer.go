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
// are emitted as USER_ENTERED. Untrusted strings cannot trigger formula execution.
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

type colSlice struct {
	startCol  int
	endCol    int
	isFormula bool
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
		sliceValues := extractSliceValues(table, s)
		opt := InputOptionRaw
		if s.isFormula {
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

func partitionColumns(table *Table, numCols int) []colSlice {
	colHasFormula := make([]bool, numCols)
	for _, r := range table.Rows {
		for colIdx := 0; colIdx < len(r) && colIdx < numCols; colIdx++ {
			if r[colIdx].IsFormula {
				colHasFormula[colIdx] = true
			}
		}
	}

	var slices []colSlice
	currStart := 0
	currFormula := colHasFormula[0]

	for c := 1; c < numCols; c++ {
		if colHasFormula[c] != currFormula {
			slices = append(slices, colSlice{
				startCol:  currStart,
				endCol:    c - 1,
				isFormula: currFormula,
			})
			currStart = c
			currFormula = colHasFormula[c]
		}
	}
	slices = append(slices, colSlice{
		startCol:  currStart,
		endCol:    numCols - 1,
		isFormula: currFormula,
	})

	return slices
}

func extractSliceValues(table *Table, s colSlice) [][]any {
	sliceValues := make([][]any, len(table.Rows))
	width := s.endCol - s.startCol + 1

	for rIdx, r := range table.Rows {
		rowVals := make([]any, width)
		for offset := range width {
			cIdx := s.startCol + offset
			if cIdx < len(r) {
				rowVals[offset] = r[cIdx].RawVal
			} else {
				rowVals[offset] = ""
			}
		}
		sliceValues[rIdx] = rowVals
	}

	return sliceValues
}
