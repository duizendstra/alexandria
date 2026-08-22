package privacyfilter

import (
	"path/filepath"
	"strings"

	"github.com/duizendstra/alexandria/go/discovery/search"
)

// Filter removes sensitive documents and redacts sensitive content.
type Filter struct {
	skipPatternsLower   []string
	redactPatterns      []string
	redactPatternsLower []string
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

	redacts := []string{
		"GITHUB_TOKEN",
		"API_KEY",
		"SECRET_KEY",
		"PRIVATE_KEY",
		"Bearer ",
		"password=",
		"token=",
	}
	redactPatternsLower := make([]string, len(redacts))
	for i, r := range redacts {
		redactPatternsLower[i] = strings.ToLower(r)
	}

	return &Filter{
		skipPatternsLower:   skipPatternsLower,
		redactPatterns:      redacts,
		redactPatternsLower: redactPatternsLower,
	}
}

// Apply filters a slice of documents, removing sensitive ones and redacting
// sensitive content from the rest. Returns clean documents and skip count.
//
// Redaction matches patterns case-insensitively and covers the title, the
// content and every metadata value, so a secret does not enter the index by
// changing case or by living in a field other than the body.
func (f *Filter) Apply(docs []search.Document) ([]search.Document, int) { //nolint:gocritic // Named returns conflict with nonamedreturns.
	var clean []search.Document
	skipped := 0

	for i := range docs {
		if f.shouldSkip(&docs[i]) {
			skipped++

			continue
		}

		// Redact sensitive content wherever it can be indexed.
		docs[i].Title = f.redact(docs[i].Title)
		docs[i].Content = f.redact(docs[i].Content)
		for k, v := range docs[i].Metadata {
			if r := f.redact(v); r != v {
				docs[i].Metadata[k] = r
			}
		}
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

// redact replaces every line that contains a redaction pattern with a marker
// naming the pattern as configured. Matching folds case on both sides; the
// folded text is only ever compared, never returned, so untouched lines keep
// their original bytes.
func (f *Filter) redact(content string) string {
	if content == "" {
		return content
	}

	contentLower := strings.ToLower(content)
	hasAny := false
	for _, patternLower := range f.redactPatternsLower {
		if strings.Contains(contentLower, patternLower) {
			hasAny = true

			break
		}
	}
	if !hasAny {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineLower := strings.ToLower(line)
		for j, patternLower := range f.redactPatternsLower {
			if strings.Contains(lineLower, patternLower) {
				lines[i] = "[REDACTED — contains " + f.redactPatterns[j] + "]"

				break // Stop checking other patterns for this line once redacted.
			}
		}
	}

	return strings.Join(lines, "\n")
}
