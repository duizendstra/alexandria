package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

// ErrInvalidJSONLine indicates a corrupted or non-JSON line encountered while streaming audit logs.
var ErrInvalidJSONLine = errors.New("invalid JSON line in audit stream")

// Reader parses a JSONL stream of audit entries line-by-line.
type Reader struct {
	scanner *bufio.Scanner
	lineNum int
}

// NewReader constructs a new streaming audit log Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Read parses and returns the next Entry from the stream.
// Returns io.EOF when the stream is exhausted.
func (r *Reader) Read() (Entry, error) {
	for r.scanner.Scan() {
		r.lineNum++
		line := r.scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines cleanly.
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return Entry{}, fmt.Errorf("%w at line %d: %w", ErrInvalidJSONLine, r.lineNum, err)
		}

		return entry, nil
	}

	if err := r.scanner.Err(); err != nil {
		return Entry{}, fmt.Errorf("read audit stream: %w", err)
	}

	return Entry{}, io.EOF
}

// ReadAll reads all remaining entries from the stream until EOF.
func (r *Reader) ReadAll() ([]Entry, error) {
	var entries []Entry
	for {
		entry, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ReadEntries reads all audit entries from the provided reader.
func ReadEntries(r io.Reader) ([]Entry, error) {
	return NewReader(r).ReadAll()
}

// Filter returns a new slice containing only the audit entries that satisfy the predicate.
func Filter(entries []Entry, predicate func(Entry) bool) []Entry {
	if predicate == nil {
		return entries
	}

	var filtered []Entry
	for _, e := range entries {
		if predicate(e) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

// defaultTopResourcesLimit is the default maximum number of top resources returned.
const defaultTopResourcesLimit = 10

// AggregateScorecard aggregates a sequence of audit entries into a summary Scorecard.
func AggregateScorecard(entries []Entry) Scorecard {
	byActor := make(map[string]int)
	byAction := make(map[string]int)
	resourceFreq := make(map[string]int)

	for _, e := range entries {
		if e.Actor != "" {
			byActor[e.Actor]++
		}
		if e.Action != "" {
			byAction[e.Action]++
		}
		if e.Resource != "" {
			resourceFreq[e.Resource]++
		}
	}

	type resCount struct {
		resource string
		count    int
	}

	resCounts := make([]resCount, 0, len(resourceFreq))
	for res, cnt := range resourceFreq {
		resCounts = append(resCounts, resCount{resource: res, count: cnt})
	}

	sort.Slice(resCounts, func(i, j int) bool {
		if resCounts[i].count != resCounts[j].count {
			return resCounts[i].count > resCounts[j].count
		}

		return resCounts[i].resource < resCounts[j].resource
	})

	limit := min(len(resCounts), defaultTopResourcesLimit)

	topResources := make([]string, limit)
	for i := range limit {
		topResources[i] = resCounts[i].resource
	}

	return Scorecard{
		Total:        len(entries),
		ByActor:      byActor,
		ByAction:     byAction,
		TopResources: topResources,
	}
}
