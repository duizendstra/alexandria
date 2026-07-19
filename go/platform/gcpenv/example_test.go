// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package gcpenv_test

import (
	"context"
	"fmt"
	"os"

	"github.com/duizendstra/alexandria/go/platform/gcpenv"
)

func ExampleFromEnv() {
	// GCP_PROJECT_ID has the highest priority of the canonical variables.
	os.Setenv("GCP_PROJECT_ID", "demo-project")
	defer os.Unsetenv("GCP_PROJECT_ID")

	fmt.Println(gcpenv.FromEnv())
	// Output: demo-project
}

func ExampleProjectID() {
	// With an environment variable set, resolution never reaches the GCE
	// metadata service; GCP_METADATA_DISABLED keeps the example hermetic
	// even without one.
	os.Setenv("GCP_PROJECT_ID", "demo-project")
	os.Setenv("GCP_METADATA_DISABLED", "true")

	defer func() {
		os.Unsetenv("GCP_PROJECT_ID")
		os.Unsetenv("GCP_METADATA_DISABLED")
	}()

	fmt.Println(gcpenv.ProjectID(context.Background()))
	// Output: demo-project
}

func ExampleMetadataDisabled() {
	os.Setenv("GCP_METADATA_DISABLED", "true")
	defer os.Unsetenv("GCP_METADATA_DISABLED")

	fmt.Println(gcpenv.MetadataDisabled())
	// Output: true
}
