# go/discovery/search/searchtest

`go/discovery/search/searchtest` provides reusable contract tests to verify that any custom `search.Index` implementation conforms to the expected behavior and semantic contracts.

## Features

- **Comprehensive Interface Check**: Validates basic document indexing, searching, and deleting.
- **Batching Lifecycle Verification**: Ensures `BatchIndex` and `Count` work correctly together.
- **Filtering Contract Enforcement**: Verifies document kind filter restrictions are properly applied.
- **Edge Case Validation**: Confirms proper handling of search operations with no results or invalid IDs.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/discovery/search/searchtest
```

## Quick Start

### Verifying a Custom Index Implementation

Add the contract test to your adapter's unit test suite:

```go
package myadapter_test

import (
	"context"
	"testing"

	"github.com/duizendstra/alexandria/go/discovery/search"
	"github.com/duizendstra/alexandria/go/discovery/search/searchtest"
)

// MyIndex represents a custom index adapter under test
type MyIndex struct {
	store map[string]search.Document
}

func (m *MyIndex) Index(ctx context.Context, doc search.Document) error {
	m.store[doc.ID] = doc
	return nil
}

func (m *MyIndex) BatchIndex(ctx context.Context, docs []search.Document) error {
	for _, doc := range docs {
		m.store[doc.ID] = doc
	}
	return nil
}

func (m *MyIndex) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *MyIndex) Count(ctx context.Context) (int, error) {
	return len(m.store), nil
}

func (m *MyIndex) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	var results []search.Result
	for _, doc := range m.store {
		results = append(results, search.Result{
			Document: doc,
			Score:    1.0,
		})
	}
	return results, nil
}

func TestMyIndexContract(t *testing.T) {
	searchtest.IndexContractTest(t, func() search.Index {
		return &MyIndex{store: make(map[string]search.Document)}
	})
}
```

## SRE & Performance Hardening details

1. **Semantic Drift Protection**: Running these contract tests in CI prevents subtle API changes or vendor-specific behavior differences from introducing bugs during production database migrations.
2. **Subtest Isolation**: Uses Go's native `t.Run` structure for test isolation, allowing test execution to recover cleanly and check other methods even if one assertion fails.
3. **Empty Results Verification**: Confirms that queries matching no files return zero-length slices instead of nil values or unhandled API errors, defending against downstream slice handling panics.
