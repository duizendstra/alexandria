// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

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
	metadataProjectID string
	metadataOnce      sync.Once
	metadataURL       = "http://metadata.google.internal/computeMetadata/v1/project/project-id"
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}
