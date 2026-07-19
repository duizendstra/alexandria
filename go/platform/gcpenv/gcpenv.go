// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

// Package gcpenv resolves the active GCP project ID from the process
// environment and, when running on managed GCP platforms, from the GCE
// metadata service.
//
// It is the canonical resolver for the Alexandria ecosystem: every module
// that needs a project ID should use this package so that the environment
// variable priority is identical everywhere.
//
// Resolution order:
//
//  1. Environment variables: GCP_PROJECT_ID, GOOGLE_CLOUD_PROJECT,
//     GCP_PROJECT, PROJECT_ID (first non-empty wins).
//  2. The GCE metadata service, unless GCP_METADATA_DISABLED=true.
//     The result is cached for the lifetime of the Resolver.
//
// An empty string is returned when the project ID cannot be determined;
// callers choose their own fallback presentation.
package gcpenv

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// envKeys is the canonical priority order for project ID environment
// variables across the Alexandria ecosystem.
//
//nolint:gochecknoglobals // Immutable canonical key order shared by all lookups.
var envKeys = []string{
	"GCP_PROJECT_ID",
	"GOOGLE_CLOUD_PROJECT",
	"GCP_PROJECT",
	"PROJECT_ID",
}

const (
	// defaultMetadataURL is the GCE metadata endpoint for the project ID.
	defaultMetadataURL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"

	// defaultMetadataTimeout keeps the probe short so non-GCP environments
	// are not blocked noticeably on first resolution.
	defaultMetadataTimeout = 500 * time.Millisecond

	// maxMetadataBodyBytes is the maximum number of bytes read from the
	// metadata response body.
	maxMetadataBodyBytes = 1024
)

// Resolver resolves a GCP project ID. The zero value is ready to use and
// probes the real GCE metadata service; tests can point MetadataURL at a
// fake server. A Resolver caches a successful metadata lookup forever.
type Resolver struct {
	// MetadataURL overrides the GCE metadata endpoint. Empty means the
	// real endpoint.
	MetadataURL string

	// Client overrides the HTTP client used for the metadata probe.
	// Nil means a client with a 500ms timeout.
	Client *http.Client

	mu     sync.Mutex
	cached string
}

// FromEnv returns the project ID from the first non-empty canonical
// environment variable, or "" when none is set.
func FromEnv() string {
	for _, key := range envKeys {
		if id := os.Getenv(key); id != "" {
			return id
		}
	}

	return ""
}

// MetadataDisabled reports whether the GCE metadata probe is disabled via
// GCP_METADATA_DISABLED=true (case-insensitive).
func MetadataDisabled() bool {
	return strings.EqualFold(os.Getenv("GCP_METADATA_DISABLED"), "true")
}

// ProjectID resolves the project ID: environment first, then the GCE
// metadata service unless disabled. It returns "" when undetectable.
func (r *Resolver) ProjectID(ctx context.Context) string {
	if id := FromEnv(); id != "" {
		return id
	}

	if MetadataDisabled() {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != "" {
		return r.cached
	}

	r.cached = r.queryMetadata(ctx)

	return r.cached
}

// queryMetadata queries the GCE metadata service for the project ID,
// returning "" on any failure.
func (r *Resolver) queryMetadata(ctx context.Context) string {
	url := r.MetadataURL
	if url == "" {
		url = defaultMetadataURL
	}

	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: defaultMetadataTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return ""
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMetadataBodyBytes))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}

//nolint:gochecknoglobals // Shared default resolver backing the package-level API.
var defaultResolver Resolver

// ProjectID resolves the project ID using the shared default Resolver.
func ProjectID(ctx context.Context) string {
	return defaultResolver.ProjectID(ctx)
}
