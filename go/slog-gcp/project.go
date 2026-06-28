// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	//nolint:gochecknoglobals // Package-level cache for metadata project ID.
	metadataProjectID string
	//nolint:gochecknoglobals // sync.Once to guard metadata resolution.
	metadataOnce sync.Once
	//nolint:gochecknoglobals // Default GCE metadata URL.
	metadataURL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"
)

// detectProjectID reads the GCP project ID from environment variables,
// then falls back to the GCE metadata service on managed GCP platforms.
func detectProjectID() string {
	// Priority 1: Environment variables (fast, overridable).
	for _, key := range []string{
		"GCP_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
		"GCP_PROJECT",
		"PROJECT_ID",
	} {
		if id := os.Getenv(key); id != "" {
			return id
		}
	}

	// Bypass metadata server check if explicitly disabled (prevents 500ms block on AWS/local).
	if strings.EqualFold(os.Getenv("GCP_METADATA_DISABLED"), "true") {
		return "unknown-project"
	}

	// Priority 2: GCE metadata service (available on Cloud Run, GKE, etc.).
	metadataOnce.Do(func() {
		metadataProjectID = queryMetadataProjectID()
	})

	if metadataProjectID != "" {
		return metadataProjectID
	}

	return "unknown-project"
}

// queryMetadataProjectID queries the GCE metadata service for the project ID.
// Uses a short timeout to avoid blocking on non-GCP environments.
func queryMetadataProjectID() string {
	client := &http.Client{Timeout: 500 * time.Millisecond} //nolint:mnd // Short timeout for non-GCP fallback.

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, http.NoBody)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}

