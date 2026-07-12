package datadiff

import "math"

// DiffColumns compares two column slices and returns the schema result.
// Nested columns are flattened to dot-notation before comparison.
// Exported for use by adapters that fetch metadata separately.
func DiffColumns(left, right []Column) SchemaResult {
	flatLeft := FlattenAll(left)
	flatRight := FlattenAll(right)

	leftByName := indexColumns(flatLeft)
	rightByName := indexColumns(flatRight)

	var result SchemaResult

	for _, c := range flatLeft {
		rc, ok := rightByName[c.Name]
		if !ok {
			result.LeftOnly = append(result.LeftOnly, c)

			continue
		}
		if c.DataType != rc.DataType {
			result.TypeDiffs = append(result.TypeDiffs, ColumnTypeDiff{
				Name:      c.Name,
				LeftType:  c.DataType,
				RightType: rc.DataType,
			})
		}
	}

	for _, c := range flatRight {
		if _, ok := leftByName[c.Name]; !ok {
			result.RightOnly = append(result.RightOnly, c)
		}
	}

	result.Match = len(result.LeftOnly) == 0 &&
		len(result.RightOnly) == 0 &&
		len(result.TypeDiffs) == 0

	return result
}

func indexColumns(cols []Column) map[string]Column {
	m := make(map[string]Column, len(cols))
	for _, c := range cols {
		m[c.Name] = c
	}

	return m
}

// DiffStats compares two sets of column stats and returns diffs.
// Exported for use by adapters that fetch stats separately.
func DiffStats(left, right []ColumnStat, tolerance float64) StatsResult {
	rightByCol := make(map[string]ColumnStat, len(right))
	for _, s := range right {
		rightByCol[s.Column] = s
	}

	var diffs []StatDiff
	for _, ls := range left {
		rs, ok := rightByCol[ls.Column]
		if !ok {
			continue
		}
		diffs = append(diffs, diffFloat(ls.Column, "Sum", ls.Sum, rs.Sum, tolerance)...)
		diffs = append(diffs, diffFloat(ls.Column, "Avg", ls.Avg, rs.Avg, tolerance)...)
		diffs = append(diffs, diffFloat(ls.Column, "Min", ls.Min, rs.Min, tolerance)...)
		diffs = append(diffs, diffFloat(ls.Column, "Max", ls.Max, rs.Max, tolerance)...)
		diffs = append(diffs, diffInt(ls.Column, "CountDistinct", ls.CountDistinct, rs.CountDistinct)...)
		diffs = append(diffs, diffInt(ls.Column, "NullCount", ls.NullCount, rs.NullCount)...)
	}

	return StatsResult{
		Match: len(diffs) == 0,
		Left:  left,
		Right: right,
		Diffs: diffs,
	}
}

func diffFloat(col, field string, l, r, tolerance float64) []StatDiff {
	if l == r {
		return nil
	}
	base := math.Max(math.Abs(l), math.Abs(r))
	if base > 0 && tolerance > 0 && math.Abs(l-r)/base <= tolerance {
		return nil
	}

	return []StatDiff{{Column: col, Field: field, Left: l, Right: r, Delta: r - l}}
}

func diffInt(col, field string, l, r int64) []StatDiff {
	if l == r {
		return nil
	}

	return []StatDiff{{Column: col, Field: field, Left: float64(l), Right: float64(r), Delta: float64(r - l)}}
}
