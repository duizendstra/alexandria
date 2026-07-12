package search

import "time"

// Kind classifies a document by its origin.
type Kind string

const (
	KindCode     Kind = "code"
	KindDoc      Kind = "doc"
	KindMemory   Kind = "memory"
	KindEtching  Kind = "etching"
	KindArtifact Kind = "artifact"
	KindProto    Kind = "proto"
	KindConfig   Kind = "config"
)

// Document is the unit of indexed content. Every piece of content —
// regardless of where it lives on disk — becomes a Document before
// entering the index.
type Document struct {
	ID        string
	Kind      Kind
	Source    string            // where this content came from (brain, kratos, workspace).
	Path      string            // file path relative to source root.
	Language  string            // go, markdown, yaml, protobuf, etc.
	Title     string            // human-readable title (extracted from content).
	Content   string            // full text content.
	Metadata  map[string]string // extensible key-value pairs.
	UpdatedAt time.Time
}

// Query describes what to search for.
type Query struct {
	Text       string   // search terms.
	Kinds      []Kind   // filter by document kind (empty = all).
	Workspaces []string // filter by workspace (empty = all).
	Limit      int      // max results (0 = default).
}

// Result is a single search hit.
type Result struct {
	Document Document
	Score    float64  // relevance score (higher = more relevant).
	Snippets []string // matching text excerpts with context.
}

// Content represents acquired external content.
type Content struct {
	URL       string
	Title     string
	Body      string // markdown.
	FetchedAt time.Time
}
