// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

// Package datastream provisions Datastream change-data-capture streams from
// PostgreSQL into BigQuery.
//
// Layer:   IaC Building Block
// Concern: How do we replicate a PostgreSQL database into BigQuery in GCP?
//
// A stream is three resources: a connection profile that says how to reach the
// source, a connection profile that says which BigQuery to write to, and the
// stream that joins them. They are three functions here because the profiles
// outlive the streams that use them — one source profile commonly feeds
// several streams, and the destination profile carries no configuration at all
// beyond its own location.
//
// The source side of a PostgreSQL stream is not created by this package. The
// publication and the replication slot are SQL objects inside the database,
// created once by whoever owns the schema; this package names them and fails
// at preview time when the name is missing, rather than pretending it made
// them.
package datastream
