package audit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duizendstra/alexandria/go/observability/audit"
	"github.com/duizendstra/alexandria/go/observability/audit/file"
)

func ExampleFileWriter_Log() {
	dir, _ := os.MkdirTemp("", "audit")
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "audit.jsonl")
	clock := func() time.Time {
		return time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	}

	w, err := file.NewFileWriter(path, file.WithClock(clock))
	if err != nil {
		panic(err)
	}
	defer func() { _ = w.Close() }()

	_ = w.Log(context.Background(), audit.Entry{
		Actor:    "agent",
		Action:   "pulls.merge",
		Resource: "OWNER/REPO#42",
	})

	data, _ := os.ReadFile(path)
	// Verify it wrote valid JSONL.
	fmt.Printf("Contains actor: %v\n", strings.Contains(string(data), `"actor":"agent"`))
	fmt.Printf("Contains action: %v\n", strings.Contains(string(data), `"action":"pulls.merge"`))
	// Output:
	// Contains actor: true
	// Contains action: true
}
