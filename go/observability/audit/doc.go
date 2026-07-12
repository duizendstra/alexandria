// Package audit provides structured append-only audit logging.
//
// # What
//
// A [Writer] interface for logging mutations as structured [Entry] records.
// Ships with [FileWriter] that writes JSONL (one JSON object per line) with
// automatic size-based rotation at [DefaultMaxLogSize] (10 MB).
//
// # Who
//
// Service handlers that perform mutations (Pull Requests, Issue updates,
// data writes). Each handler accepts a Writer via an option function
// (e.g., pulls.WithAuditWriter). Pipeline operators review the audit log
// for compliance and debugging.
//
// # When
//
// Log every mutation — create, update, delete, merge. Read operations
// are not audited. The actor is extracted from the X-Dui-Actor HTTP header,
// defaulting to "api" if absent.
//
// # Where
//
// Injected into Connect RPC handlers via option functions. The JSONL output
// is committed to the repository for git-tracked traceability.
//
//	handler → audit.Writer.Log → audit.jsonl (JSONL)
//
// # Why
//
// Mutations to external systems (GitHub, vendor APIs) must be traceable.
// JSONL is grep-friendly, diff-friendly, and git-friendly. The append-only
// contract prevents tampering. Size-based rotation keeps logs manageable.
//
// Domain:  Observability
// Concern: How do we audit-log mutations?
package audit
