# go/discovery/privacyfilter

`go/discovery/privacyfilter` acts as an ingestion-phase privacy guard for workspace indexing, preventing sensitive files and credentials from entering document search indices.

## Features

- **Pre-Index Filtering**: Skips sensitive files (e.g., `.env`, key/credential files, SSH private keys) based on file name or path.
- **Line-Level Redaction**: Detects and redacts lines containing keys and tokens (e.g., `GITHUB_TOKEN`, `API_KEY`, `password=`) inside non-sensitive documents. Matching is case-insensitive (`PASSWORD=` and `password=` both redact) and covers the title, the content and every metadata value.
- **Fast-Path String Verification**: Bypasses costly string manipulation or splitting for clean files.
- **Low Overhead**: Pure, dependency-free implementation using optimized Go standard libraries.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/discovery/privacyfilter
```

## Quick Start

### Filtering and Redacting Indexed Documents

```go
package main

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/discovery/privacyfilter"
	"github.com/duizendstra/alexandria/go/discovery/search"
)

func main() {
	filter := privacyfilter.New()

	docs := []search.Document{
		{
			ID:      "doc:1",
			Path:    "src/main.go",
			Content: "package main\n\nconst token = \"123\"\n// TODO: replace with proper configuration",
		},
		{
			ID:      "doc:2",
			Path:    "configs/prod.env",
			Content: "DATABASE_PASSWORD=supersecret",
		},
		{
			ID:      "doc:3",
			Path:    "src/db.go",
			Content: "package main\n\nvar dbURL = \"postgres://localhost:5432\"",
		},
	}

	// Apply redaction rules and filter out matching files
	cleaned, skippedCount := filter.Apply(docs)

	fmt.Printf("Ingested: %d, Skipped: %d\n", len(cleaned), skippedCount)
	for _, doc := range cleaned {
		fmt.Printf("ID: %s, Path: %s, Content:\n%s\n", doc.ID, doc.Path, doc.Content)
	}
}
```

## SRE & Performance Hardening details

1. **Bypass Allocations for Clean Content**: Scans a case-folded copy of each field using `strings.Contains` to check if a redacting pattern is present. If none are found, the field is returned directly, avoiding any allocation overhead of `strings.Split`; the folded copy is only compared, never returned, so clean fields keep their original bytes.
2. **Early Exclusion logic**: Evaluates file names and path strings before scanning file contents, enabling instant exclusion of known sensitive paths (like `.pem`, `.env`, and SSH private keys) without reading their contents.
3. **Deterministic Replacement Output**: Replaces matching secret lines with a standardized `[REDACTED — contains PATTERN]` notice, stripping credentials while retaining context.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Indexing pipelines**: anything about to put workspace content into a
  searchable store.
- **Content ingestion adapters**: code reading files or documents from a
  source that was never curated for indexing.

### Load-Bearing Promises
1. **The Gate Is Before The Index, Not After**: filtering runs prior to
   indexing rather than as post-hoc triage, so sensitive content never enters
   the store in the first place.
2. **Sensitive Files Are Skipped Whole**: `.env` files, credentials and key
   material are excluded entirely rather than redacted field by field.
3. **Redaction Covers Every Surface**: title, body and metadata values are all
   redacted — not the body alone.
4. **Matching Is Case-Insensitive**: a case variant of a sensitive pattern is
   redacted like the canonical spelling.
5. **Clean Content Is Returned Unchanged**: a document with no match comes back
   identical, so the filter is safe to run over everything.
