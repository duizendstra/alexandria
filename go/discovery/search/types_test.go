// Domain:  Discovery
// Concern: Do the search types correctly model workspace content?
package search_test

import (
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

func TestDocumentKinds(t *testing.T) {
	kinds := []search.Kind{
		search.KindCode,
		search.KindDoc,
		search.KindMemory,
		search.KindEtching,
		search.KindArtifact,
		search.KindProto,
		search.KindConfig,
	}

	for _, k := range kinds {
		if k == "" {
			t.Error("Kind constant must not be empty")
		}
	}
	t.Logf("All %d Kind constants are valid", len(kinds))
}

func TestDocumentCreation(t *testing.T) {
	doc := search.Document{
		ID:        "test:1",
		Kind:      search.KindCode,
		Source:    "workspace",
		Path:      "main.go",
		Language:  "go",
		Title:     "Main package",
		Content:   "package main",
		Metadata:  map[string]string{"key": "val"},
		UpdatedAt: time.Now(),
	}

	if doc.ID == "" {
		t.Error("Document ID must not be empty")
	}
	if doc.Kind != search.KindCode {
		t.Errorf("expected KindCode, got %s", doc.Kind)
	}
	if doc.Metadata["key"] != "val" {
		t.Error("metadata not preserved")
	}
}

func TestQueryDefaults(t *testing.T) {
	q := search.Query{Text: "test"}

	if q.Limit != 0 {
		t.Error("default limit should be 0 (meaning: adapter decides)")
	}
	if len(q.Kinds) != 0 {
		t.Error("default kinds should be empty (meaning: all)")
	}
}

func TestResultScoring(t *testing.T) {
	r := search.Result{
		Document: search.Document{ID: "1"},
		Score:    0.85,
		Snippets: []string{"matching text"},
	}

	if r.Score < 0 || r.Score > 1 {
		t.Errorf("score should be 0-1, got %.2f", r.Score)
	}
	if len(r.Snippets) == 0 {
		t.Error("snippets should not be empty")
	}
}
