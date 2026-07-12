package privacyfilter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/discovery/privacyfilter"
	"github.com/duizendstra/alexandria/go/discovery/search"
)

func TestSkipsSensitiveFiles(t *testing.T) {
	f := privacyfilter.New()

	docs := []search.Document{
		{ID: "1", Path: "main.go", Content: "package main"},
		{ID: "2", Path: ".env", Content: "SECRET=foo"},
		{ID: "3", Path: "config/credentials.json", Content: "{\"key\": \"val\"}"},
		{ID: "4", Path: "docs/README.md", Content: "# Readme"},
		{ID: "5", Path: ".kratos/config.yaml", Content: "token: abc"},
		{ID: "6", Path: "keys/id_rsa", Content: "-----BEGIN RSA PRIVATE KEY-----"},
	}

	clean, skipped := f.Apply(docs)

	if skipped != 4 {
		t.Errorf("expected 4 skipped, got %d", skipped)
	}
	if len(clean) != 2 {
		t.Errorf("expected 2 clean docs, got %d", len(clean))
	}

	for _, doc := range clean {
		t.Logf("  kept: %s", doc.Path)
	}
	t.Logf("Skipped %d sensitive files", skipped)
}

func TestRedactsSensitiveContent(t *testing.T) {
	f := privacyfilter.New()

	docs := []search.Document{
		{
			ID:        "1",
			Path:      "deploy.sh",
			Content:   "#!/bin/bash\nexport GITHUB_TOKEN=ghp_abc123\necho deploy",
			UpdatedAt: time.Now(),
		},
	}

	clean, _ := f.Apply(docs)

	if len(clean) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(clean))
	}

	if !containsRedacted(clean[0].Content) {
		t.Errorf("expected redacted content, got: %s", clean[0].Content)
	}

	t.Logf("Redacted content:\n%s", clean[0].Content)
}

func TestKeepsCleanContent(t *testing.T) {
	f := privacyfilter.New()

	docs := []search.Document{
		{ID: "1", Path: "types.go", Content: "type Document struct {\n\tID string\n}"},
	}

	clean, skipped := f.Apply(docs)

	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
	if clean[0].Content != docs[0].Content {
		t.Error("clean content should not modify safe documents")
	}
}

func containsRedacted(s string) bool {
	return strings.Contains(s, "[REDACTED")
}
