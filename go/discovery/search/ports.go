package search

import "context"

// Index is the output port — store and search documents.
type Index interface {
	// Index adds or updates a single document.
	Index(ctx context.Context, doc Document) error

	// BatchIndex adds or updates multiple documents.
	BatchIndex(ctx context.Context, docs []Document) error

	// Search returns documents matching the query, ordered by relevance.
	Search(ctx context.Context, query Query) ([]Result, error)

	// Delete removes a document by ID.
	Delete(ctx context.Context, id string) error

	// Count returns the total number of indexed documents.
	Count(ctx context.Context) (int, error)
}

// ContentSource is the input port — scan a storage location for documents.
// Each adapter scans one content source (brain dirs, workspace files, etc).
type ContentSource interface {
	// Name returns a human-readable name for this source.
	Name() string

	// Scan returns all documents from this source.
	Scan(ctx context.Context) ([]Document, error)
}

// ContentFetcher is the input port — acquire content from external URLs.
type ContentFetcher interface {
	// Fetch retrieves content from a URL and converts it to markdown.
	Fetch(ctx context.Context, url string) (Content, error)
}
