// Package sheets provides high-level, production-grade Google Sheets automation
// and data export capabilities for the Alexandria ecosystem.
//
// # Architecture & Problem Solved
//
// Interacting with the low-level Google Sheets v4 API directly in Go requires
// extensive boilerplate, manual batch request construction, and careful handling
// of security and rate limits. This package encapsulates four essential patterns:
//
// 1. Dual-Mode Value Partitioning (Formula Injection Protection):
// Untrusted inputs (user names, filenames, identifiers, comments) are guaranteed
// to be written with ValueInputOption("RAW"), preventing CSV / Formula Injection
// attacks (e.g. `=cmd|' /C calc'!A0`). Only cells explicitly marked as formulas
// (e.g. `Formula(...)` or `Hyperlink(...)`) are routed to ValueInputOption("USER_ENTERED").
//
// 2. Idempotent Tab Synchronization (The Projection Pattern):
// In data pipelines and migration tools (such as wave planning, link sheets, and
// access reviews), spreadsheet tabs act as read projections of authoritative database
// truth. The Syncer resolves tabs by GID or Title, expands grid boundaries as needed,
// clears old ranges, updates values in batched operations, and freezes header rows.
//
// 3. Declarative Themes & Professional Styling:
// Pre-configured themes (CorporateNavy, ModernSlate, EmeraldForest, CleanMinimal)
// apply unified visual standards across all organization sheets, including frozen
// colored header bars, contrasting bold text, alternating row zebra banding,
// auto-fit column resizing, and custom pixel-width column overrides.
//
// 4. Drive Integration & Folder Placement:
// Document creation automatically supports target Drive folder placement
// (using Drive v3 AddParents / RemoveParents) and returns direct browser URLs.
package sheets
