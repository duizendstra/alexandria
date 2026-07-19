// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package gcpenv_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/gcpenv"
)

// clearEnv unsets every canonical project ID variable for the test.
func clearEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{"GCP_PROJECT_ID", "GOOGLE_CLOUD_PROJECT", "GCP_PROJECT", "PROJECT_ID", "GCP_METADATA_DISABLED"} {
		t.Setenv(key, "")
	}
}

func TestFromEnvPriority(t *testing.T) {
	clearEnv(t)
	t.Setenv("PROJECT_ID", "low")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "mid")
	t.Setenv("GCP_PROJECT_ID", "high")

	if got := gcpenv.FromEnv(); got != "high" {
		t.Fatalf("FromEnv() = %q, want %q", got, "high")
	}
}

func TestFromEnvEmpty(t *testing.T) {
	clearEnv(t)

	if got := gcpenv.FromEnv(); got != "" {
		t.Fatalf("FromEnv() = %q, want empty", got)
	}
}

func TestProjectIDEnvWinsOverMetadata(t *testing.T) {
	clearEnv(t)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "env-project")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("metadata-project"))
	}))
	defer server.Close()

	r := &gcpenv.Resolver{MetadataURL: server.URL}
	if got := r.ProjectID(t.Context()); got != "env-project" {
		t.Fatalf("ProjectID() = %q, want %q", got, "env-project")
	}
}

func TestProjectIDMetadataFallbackAndCache(t *testing.T) {
	clearEnv(t)

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			t.Errorf("missing Metadata-Flavor header")
		}

		calls.Add(1)
		_, _ = w.Write([]byte("metadata-project\n"))
	}))
	defer server.Close()

	r := &gcpenv.Resolver{MetadataURL: server.URL}

	for range 3 {
		if got := r.ProjectID(t.Context()); got != "metadata-project" {
			t.Fatalf("ProjectID() = %q, want %q", got, "metadata-project")
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("metadata calls = %d, want 1 (cached)", got)
	}
}

func TestProjectIDMetadataDisabled(t *testing.T) {
	clearEnv(t)
	t.Setenv("GCP_METADATA_DISABLED", "TRUE")

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("metadata server must not be called when disabled")
	}))
	defer server.Close()

	r := &gcpenv.Resolver{MetadataURL: server.URL}
	if got := r.ProjectID(t.Context()); got != "" {
		t.Fatalf("ProjectID() = %q, want empty", got)
	}
}

func TestProjectIDMetadataNon200(t *testing.T) {
	clearEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := &gcpenv.Resolver{MetadataURL: server.URL}
	if got := r.ProjectID(t.Context()); got != "" {
		t.Fatalf("ProjectID() = %q, want empty", got)
	}
}

func TestProjectIDMetadataUnreachable(t *testing.T) {
	clearEnv(t)

	// Closed server: connection refused, resolver must return "".
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	server.Close()

	r := &gcpenv.Resolver{MetadataURL: server.URL}
	if got := r.ProjectID(t.Context()); got != "" {
		t.Fatalf("ProjectID() = %q, want empty", got)
	}
}

func TestMetadataDisabled(t *testing.T) {
	clearEnv(t)

	if gcpenv.MetadataDisabled() {
		t.Fatal("MetadataDisabled() = true, want false")
	}

	t.Setenv("GCP_METADATA_DISABLED", "true")

	if !gcpenv.MetadataDisabled() {
		t.Fatal("MetadataDisabled() = false, want true")
	}
}
