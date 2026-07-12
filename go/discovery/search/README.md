# go/discovery/search

`go/discovery/search` defines the core interfaces, data structures, and types for building document search, indexing, scoring, and text extraction adapters inside the Alexandria platform.

## Features

- **Port-Adapter Architecture**: Standardized interfaces (`Index`, `ContentSource`, `ContentFetcher`) decouple domain logic from database engines.
- **Unified Document Model**: A structured representation of content (`Document`) classifying kinds like code, documents, artifacts, or memories.
- **Type-Safe Filtering**: Advanced search `Query` configuration with strongly-typed kind arrays, workspace filters, and result limit bounds.
- **Zero External Dependencies**: Pure Go library that runs in any environment without platform-specific database libraries.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/discovery/search
```

## Quick Start

### Implementing and Using a Custom Search Index

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

// memoryIndex is a simple in-memory search.Index implementation for illustration
type memoryIndex struct {
	docs map[string]search.Document
}

func (m *memoryIndex) Index(ctx context.Context, doc search.Document) error {
	m.docs[doc.ID] = doc
	return nil
}

func (m *memoryIndex) BatchIndex(ctx context.Context, docs []search.Document) error {
	for _, doc := range docs {
		m.docs[doc.ID] = doc
	}
	return nil
}

func (m *memoryIndex) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	var results []search.Result
	for _, doc := range m.docs {
		if strings.Contains(strings.ToLower(doc.Content), strings.ToLower(q.Text)) {
			results = append(results, search.Result{
				Document: doc,
				Score:    1.0,
			})
		}
	}
	return results, nil
}

func (m *memoryIndex) Delete(ctx context.Context, id string) error {
	delete(m.docs, id)
	return nil
}

func (m *memoryIndex) Count(ctx context.Context) (int, error) {
	return len(m.docs), nil
}

func main() {
	ctx := context.Background()
	idx := &memoryIndex{docs: make(map[string]search.Document)}

	// Register a new Document
	doc := search.Document{
		ID:        "intro",
		Kind:      search.KindDoc,
		Source:    "internal",
		Path:      "docs/intro.md",
		Title:     "Introduction to Alexandria",
		Content:   "Alexandria is a resilient knowledge index module.",
		UpdatedAt: time.Now(),
	}

	if err := idx.Index(ctx, doc); err != nil {
		log.Fatalf("Failed to index: %v", err)
	}

	// Query the document index
	results, err := idx.Search(ctx, search.Query{Text: "knowledge", Limit: 10})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Search returned %d result(s):\n", len(results))
	for _, r := range results {
		fmt.Printf(" - [%s] %s (Score: %.2f)\n", r.Document.ID, r.Document.Title, r.Score)
	}
}
```

## SRE & Performance Hardening details

1. **Batch API Support**: The `BatchIndex` port contract ensures that performance-sensitive ingestion pipelines can load large datasets concurrently, reducing operational overhead and DB connection cycles.
2. **Platform Portability**: Zero platform-specific package dependencies prevent dependency locks, enabling the search layer to operate seamlessly across serverless, containerized, or local environments.
3. **Structured Query Constraints**: By enforcing query constraints inside a structured `Query` configuration, implementations protect backend engines from search parameter overflow or query injection attacks.
