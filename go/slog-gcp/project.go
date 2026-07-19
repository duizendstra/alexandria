// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"context"

	"github.com/duizendstra/alexandria/go/platform/gcpenv"
)

//nolint:gochecknoglobals // Swappable resolver so tests can fake the metadata server.
var projectResolver = &gcpenv.Resolver{}

// detectProjectID resolves the GCP project ID via the shared gcpenv
// resolver (environment variables first, then the GCE metadata service
// unless GCP_METADATA_DISABLED=true). It falls back to "unknown-project"
// so log attribution never emits an empty project segment.
func detectProjectID() string {
	if id := projectResolver.ProjectID(context.Background()); id != "" {
		return id
	}

	return "unknown-project"
}
