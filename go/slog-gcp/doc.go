// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

// Package sloggcp provides a [slog.Handler] decorator for GCP Cloud Logging.
//
// # What
//
// A [slog.Handler] wrapper that auto-injects event_id, GCP trace URL,
// span ID, and trace_sampled into every log record. Includes severity
// value mapping and Cloud Error Reporting helpers.
//
// # Who
//
// Any Go service running on GCP Cloud Run (or anywhere writing structured
// JSON to stdout for Cloud Logging ingestion).
//
// # When
//
// Call [Setup] once in main(). Then every slog call automatically
// includes GCP Cloud Logging fields with zero effort.
//
// # Where
//
// github.com/duizendstra/alexandria/go/slog-gcp
//
// # Why
//
// GCP Cloud Logging expects specific JSON field names and value
// formats. This handler bridges Go's slog to Cloud Logging's schema
// so logs are searchable, traceable, and feed Error Reporting.
package sloggcp
