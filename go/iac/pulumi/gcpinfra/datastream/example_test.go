// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datastream_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datastream"
)

func ExampleStreamConfig_Validate() {
	c := datastream.StreamConfig{
		StreamID:        "app-cdc",
		DisplayName:     "App CDC",
		Location:        "europe-west4",
		Publication:     "app_pub",
		ReplicationSlot: "app_slot",
		DatasetID:       "app_landing",
		DataFreshness:   "0s",
		DesiredState:    datastream.StateRunning,
		BackfillAll:     true,
	}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}

// A source profile takes its password either literally or by naming a Secret
// Manager version, never both: a profile carrying two credentials has an
// effective one nobody can read off the page.
func ExamplePostgresProfileConfig_Validate() {
	both := datastream.PostgresProfileConfig{
		ID:                          "source",
		DisplayName:                 "Source",
		Location:                    "europe-west4",
		Hostname:                    "198.51.100.10",
		Username:                    "replicator",
		Database:                    "app",
		Password:                    "hunter2",
		SecretManagerStoredPassword: secretVersion,
	}
	fmt.Println(both.Validate())
	// Output:
	// datastream: profile carries both Password and SecretManagerStoredPassword: "source"
}
