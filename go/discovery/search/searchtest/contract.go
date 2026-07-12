package searchtest

import (
	"context"
	"testing"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

// IndexContractTest validates that any Index implementation satisfies the port contract.
func IndexContractTest(t *testing.T, newIndex func() search.Index) {
	t.Helper()
	ctx := context.Background()

	t.Run("Index and Search", func(t *testing.T) {
		idx := newIndex()
		doc := search.Document{
			ID:      "contract:1",
			Kind:    search.KindDoc,
			Source:  "test",
			Title:   "Contract Test Document",
			Content: "This validates the Index port contract.",
		}

		if err := idx.Index(ctx, doc); err != nil {
			t.Fatalf("Index: %v", err)
		}

		results, err := idx.Search(ctx, search.Query{Text: "contract", Limit: 5}) //nolint:mnd // Clear from context.
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Search should find indexed document")
		}
	})

	t.Run("BatchIndex and Count", func(t *testing.T) {
		idx := newIndex()
		docs := []search.Document{
			{ID: "batch:1", Content: "first document about testing"},
			{ID: "batch:2", Content: "second document about testing"},
		}

		if err := idx.BatchIndex(ctx, docs); err != nil {
			t.Fatalf("BatchIndex: %v", err)
		}

		count, err := idx.Count(ctx)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		// Count should be >= 2 (grep returns -1, which is acceptable).
		if count != -1 && count < 2 {
			t.Errorf("Count should be >= 2 or -1, got %d", count)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		idx := newIndex()
		_ = idx.Index(ctx, search.Document{ID: "del:1", Content: "to delete"})

		if err := idx.Delete(ctx, "del:1"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	})

	t.Run("Empty Search", func(t *testing.T) {
		idx := newIndex()
		results, err := idx.Search(ctx, search.Query{Text: "xyznonexistent999", Limit: 5}) //nolint:mnd // Clear from context.
		if err != nil {
			t.Fatalf("Search should not error on empty: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("Search with Kind filter", func(t *testing.T) {
		idx := newIndex()
		_ = idx.BatchIndex(ctx, []search.Document{
			{ID: "kf:1", Kind: search.KindCode, Content: "func contract test code"},
			{ID: "kf:2", Kind: search.KindDoc, Content: "contract test documentation"},
		})

		results, _ := idx.Search(ctx, search.Query{
			Text:  "contract",
			Kinds: []search.Kind{search.KindCode},
		})

		for _, r := range results { //nolint:gocritic // Acceptable for this pattern.
			if r.Document.Kind != search.KindCode {
				t.Errorf("kind filter violated: got %s, want code", r.Document.Kind)
			}
		}
	})
}
