package privacyfilter

import (
	"path/filepath"
	"strings"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

// Filter removes sensitive documents and redacts sensitive content.
type Filter struct {
	skipPatternsLower []string
	redactPatterns    []string
}

// New creates a privacy filter with default patterns.
func New() *Filter {
	skips := []string{
		".env",
		"credentials",
		"secret",
		"token",
		"password",
		"private_key",
		"id_rsa",
		".pem",
		".key",
		".kratos/",
	}
	skipPatternsLower := make([]string, len(skips))
	for i, s := range skips {
		skipPatternsLower[i] = strings.ToLower(s)
	}

	return &Filter{
		skipPatternsLower: skipPatternsLower,
		redactPatterns: []string{
			"GITHUB_TOKEN",
			"API_KEY",
			"SECRET_KEY",
			"PRIVATE_KEY",
			"Bearer ",
			"password=",
			"token=",
		},
	}
}

// Apply filters a slice of documents, removing sensitive ones and redacting
// sensitive content from the rest. Returns clean documents and skip count.
func (f *Filter) Apply(docs []search.Document) ([]search.Document, int) { //nolint:gocritic // Named returns conflict with nonamedreturns.
	var clean []search.Document
	skipped := 0

	for i := range docs {
		if f.shouldSkip(&docs[i]) {
			skipped++

			continue
		}

		// Redact sensitive content.
		docs[i].Content = f.redact(docs[i].Content)
		clean = append(clean, docs[i])
	}

	return clean, skipped
}

func (f *Filter) shouldSkip(doc *search.Document) bool {
	pathLower := strings.ToLower(doc.Path)
	baseLower := strings.ToLower(filepath.Base(doc.Path))

	for _, patternLower := range f.skipPatternsLower {
		if strings.Contains(pathLower, patternLower) {
			return true
		}
		if strings.Contains(baseLower, patternLower) {
			return true
		}
	}

	return false
}

func (f *Filter) redact(content string) string {
	if content == "" {
		return content
	}

	hasAny := false
	for _, pattern := range f.redactPatterns {
		if strings.Contains(content, pattern) {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for _, pattern := range f.redactPatterns {
			if strings.Contains(line, pattern) {
				lines[i] = "[REDACTED — contains " + pattern + "]"
				break // Stop checking other patterns for this line once redacted
			}
		}
	}

	return strings.Join(lines, "\n")
}
