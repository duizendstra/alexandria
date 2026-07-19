// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package privacyfilter_test

import (
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/discovery/privacyfilter"
	"github.com/duizendstra/alexandria/go/discovery/search"
)

// FuzzApplyRedaction pins the redaction contract for arbitrary document
// content: line count is preserved, redaction is idempotent, and no
// sensitive marker survives on a non-redacted line.
func FuzzApplyRedaction(f *testing.F) {
	f.Add("plain text\nno secrets here")
	f.Add("GITHUB_TOKEN=abc123\nsafe line")
	f.Add("Authorization: Bearer eyJhbGciOi\npassword=hunter2")
	f.Add("")
	f.Add("\n\n\n")
	f.Add("API_KEY")

	patterns := []string{
		"GITHUB_TOKEN", "API_KEY", "SECRET_KEY", "PRIVATE_KEY",
		"Bearer ", "password=", "token=",
	}

	f.Fuzz(func(t *testing.T, content string) {
		filter := privacyfilter.New()

		docs := []search.Document{{Path: "notes/doc.md", Content: content}}
		clean, skipped := filter.Apply(docs)

		if skipped != 0 || len(clean) != 1 {
			t.Fatalf("Apply skipped %d of 1 docs with a clean path", skipped)
		}

		got := clean[0].Content

		// Line structure is preserved: redaction replaces lines 1:1.
		if content != "" && strings.Count(got, "\n") != strings.Count(content, "\n") {
			t.Fatalf("line count changed: %d newlines in, %d out",
				strings.Count(content, "\n"), strings.Count(got, "\n"))
		}

		// Every line either kept its original text (no pattern present) or
		// became a redaction marker.
		gotLines := strings.Split(got, "\n")
		inLines := strings.Split(content, "\n")
		for i, line := range gotLines {
			redacted := strings.HasPrefix(line, "[REDACTED — contains ")
			if redacted {
				continue
			}
			if line != inLines[i] {
				t.Fatalf("line %d rewritten without redaction: %q -> %q", i, inLines[i], line)
			}
			for _, p := range patterns {
				if strings.Contains(line, p) {
					t.Fatalf("line %d still contains %q after Apply: %q", i, p, line)
				}
			}
		}

		// Idempotence: redacting already-redacted content is a no-op.
		again, _ := filter.Apply([]search.Document{{Path: "notes/doc.md", Content: got}})
		if again[0].Content != got {
			t.Fatalf("redaction not idempotent:\n first: %q\nsecond: %q", got, again[0].Content)
		}
	})
}
