// Package file provides file-based implementations of the audit.Writer interface.
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/duizendstra/alexandria/go/observability/audit"
)

const (
	// fileMode is the permission mode for the audit log file.
	fileMode = 0o644

	// DefaultMaxLogSize is the default maximum audit log size before rotation (10 MB).
	DefaultMaxLogSize = 10 << 20
)

// FileWriter writes JSONL audit entries to a file. Safe for concurrent use.
// Supports size-based log rotation.
type FileWriter struct {
	mu         sync.Mutex
	file       *os.File
	path       string
	clock      func() time.Time
	maxLogSize int64
	written    int64
}

// Option configures a FileWriter.
type Option func(*FileWriter)

// WithClock sets a custom clock (for testing).
func WithClock(fn func() time.Time) Option {
	return func(w *FileWriter) {
		w.clock = fn
	}
}

// WithMaxLogSize sets the maximum log size before rotation.
func WithMaxLogSize(size int64) Option {
	return func(w *FileWriter) {
		w.maxLogSize = size
	}
}

// NewFileWriter creates an append-only audit log writer at the given path.
// The parent directory must exist.
func NewFileWriter(path string, opts ...Option) (*FileWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode) //nolint:gosec // path from server config
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}

	// Get current file size for rotation tracking.
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("stat audit log: %w", err)
	}

	w := &FileWriter{
		file:       f,
		path:       path,
		clock:      time.Now,
		maxLogSize: DefaultMaxLogSize,
		written:    info.Size(),
	}

	for _, o := range opts {
		o(w)
	}

	return w, nil
}

// Log appends an audit entry. The timestamp is set automatically.
// If the log exceeds the maximum size, it is rotated before writing.
func (w *FileWriter) Log(_ context.Context, entry audit.Entry) error {
	entry.Time = w.clock().Format(time.RFC3339)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.written >= w.maxLogSize {
		if err := w.rotateLocked(); err != nil {
			return fmt.Errorf("rotate audit log: %w", err)
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}

	// Write entry + newline as a single operation.
	data = append(data, '\n')

	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}

	w.written += int64(n)

	return nil
}

// Close closes the underlying file in a synchronized manner.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close audit log: %w", err)
	}

	return nil
}

// rotateLocked renames the current log to {path}.1 and opens a new file.
// Must be called with w.mu held.
func (w *FileWriter) rotateLocked() error {
	_ = w.file.Close()

	backupPath := w.path + ".1"

	// Remove old backup if it exists (keep only 1 backup).
	_ = os.Remove(backupPath)

	if err := os.Rename(w.path, backupPath); err != nil {
		// Fallback self-healing: reopen original file in append mode to maintain capacity.
		f, reopenErr := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode)
		if reopenErr == nil {
			w.file = f
		}

		return fmt.Errorf("rename audit log: %w", err)
	}

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		// Fallback self-healing: reopen backupPath in append mode to keep logging going on backup file.
		fb, backupReopenErr := os.OpenFile(backupPath, os.O_APPEND|os.O_WRONLY, fileMode) //nolint:gosec // path is trusted
		if backupReopenErr == nil {
			w.file = fb
		}

		return fmt.Errorf("create new audit log: %w", err)
	}

	w.file = f
	w.written = 0

	return nil
}

// maxTopResources is the number of most-touched resources to report.
const maxTopResources = 5

// ReadScorecard reads the audit log at path and returns a summary.
// Uses stream decoding with json.Decoder to handle arbitrarily long lines without buffer errors.
func ReadScorecard(path string) (audit.Scorecard, error) {
	f, err := os.Open(path) //nolint:gosec // path is trusted (from config)
	if err != nil {
		return audit.Scorecard{}, fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only; close failure is harmless

	sc := audit.Scorecard{
		ByActor:  make(map[string]int),
		ByAction: make(map[string]int),
	}
	resources := make(map[string]int)

	dec := json.NewDecoder(f)
	for dec.More() {
		var e audit.Entry
		if err := dec.Decode(&e); err != nil {
			continue // skip malformed entries.
		}

		sc.Total++
		sc.ByActor[e.Actor]++
		sc.ByAction[e.Action]++
		resources[e.Resource]++
	}

	sc.TopResources = topN(resources, maxTopResources)

	return sc, nil
}

// topN returns the top n keys by count, formatted as "key (count)".
func topN(m map[string]int, n int) []string {
	type kv struct {
		key   string
		count int
	}

	pairs := make([]kv, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	if len(pairs) > n {
		pairs = pairs[:n]
	}

	result := make([]string, len(pairs))
	for i, p := range pairs {
		result[i] = fmt.Sprintf("%s (%d)", p.key, p.count)
	}

	return result
}
