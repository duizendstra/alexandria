// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

// Package cloudsql provisions Cloud SQL for PostgreSQL instances, their
// databases and their users in Google Cloud.
//
// Layer:   IaC Building Block
// Concern: How do we create a Cloud SQL PostgreSQL server, the databases on
// it, and the identities allowed to connect to it?
//
// The three concerns are three functions rather than one, because the caller
// owns their order and their fan-out: one instance carries many databases and
// many users, and a stack routinely adds a database without touching the
// server. ApplyDatabases and ApplyUsers take the instance name that
// ApplyInstance returns, so the dependency is expressed in the value, not in
// a comment.
package cloudsql
