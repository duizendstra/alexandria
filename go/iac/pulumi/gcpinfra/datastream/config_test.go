// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datastream_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datastream"
)

const (
	region = "europe-west4"

	caseMissingID          = "missing id"
	caseMissingDisplayName = "missing display name"
	caseMissingLocation    = "missing location"
	secretVersion          = "projects/p/secrets/s/versions/1"
	sinkID                 = "sink"
	sinkDisplayName        = "Sink"
)

func validSource() datastream.PostgresProfileConfig {
	return datastream.PostgresProfileConfig{
		ID:          "source",
		DisplayName: "Source",
		Location:    region,
		Hostname:    "198.51.100.10",
		Username:    "replicator",
		Database:    "app",
		Password:    "p",
	}
}

func validStream() datastream.StreamConfig {
	return datastream.StreamConfig{
		StreamID:        "app-cdc",
		DisplayName:     "App CDC",
		Location:        region,
		Publication:     "app_pub",
		ReplicationSlot: "app_slot",
		DatasetID:       "app_landing",
	}
}

func TestPostgresProfileValidateValid(t *testing.T) {
	t.Parallel()

	if err := validSource().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPostgresProfileValidateRejectsIncomplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mut  func(*datastream.PostgresProfileConfig)
		want error
	}{
		{caseMissingID, func(c *datastream.PostgresProfileConfig) { c.ID = "" }, datastream.ErrProfileIDRequired},
		{
			caseMissingDisplayName,
			func(c *datastream.PostgresProfileConfig) { c.DisplayName = "" },
			datastream.ErrDisplayNameRequired,
		},
		{caseMissingLocation, func(c *datastream.PostgresProfileConfig) { c.Location = "" }, datastream.ErrLocationRequired},
		{"missing hostname", func(c *datastream.PostgresProfileConfig) { c.Hostname = "" }, datastream.ErrHostnameRequired},
		{"missing username", func(c *datastream.PostgresProfileConfig) { c.Username = "" }, datastream.ErrUsernameRequired},
		{"missing database", func(c *datastream.PostgresProfileConfig) { c.Database = "" }, datastream.ErrDatabaseRequired},
		{"port too high", func(c *datastream.PostgresProfileConfig) { c.Port = 70000 }, datastream.ErrPortInvalid},
		{"negative port", func(c *datastream.PostgresProfileConfig) { c.Port = -1 }, datastream.ErrPortInvalid},
		{
			"no credential",
			func(c *datastream.PostgresProfileConfig) { c.Password = "" },
			datastream.ErrCredentialRequired,
		},
		{
			"two credentials",
			func(c *datastream.PostgresProfileConfig) {
				c.SecretManagerStoredPassword = secretVersion
			},
			datastream.ErrCredentialAmbiguous,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validSource()
			tt.mut(&c)

			if err := c.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

// A profile that reads its password from Secret Manager carries no literal
// password, which is the whole point of that field.
func TestPostgresProfileValidateAcceptsSecretManagerCredential(t *testing.T) {
	t.Parallel()

	c := validSource()
	c.Password = ""
	c.SecretManagerStoredPassword = secretVersion

	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBigQueryProfileValidate(t *testing.T) {
	t.Parallel()

	valid := datastream.BigQueryProfileConfig{ID: sinkID, DisplayName: sinkDisplayName, Location: region}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	tests := []struct {
		name string
		cfg  datastream.BigQueryProfileConfig
		want error
	}{
		{
			caseMissingID,
			datastream.BigQueryProfileConfig{DisplayName: sinkDisplayName, Location: region},
			datastream.ErrProfileIDRequired,
		},
		{
			caseMissingDisplayName,
			datastream.BigQueryProfileConfig{ID: sinkID, Location: region},
			datastream.ErrDisplayNameRequired,
		},
		{
			caseMissingLocation,
			datastream.BigQueryProfileConfig{ID: sinkID, DisplayName: sinkDisplayName},
			datastream.ErrLocationRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.cfg.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestStreamValidateValid(t *testing.T) {
	t.Parallel()

	if err := validStream().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStreamValidateRejectsIncomplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mut  func(*datastream.StreamConfig)
		want error
	}{
		{caseMissingID, func(c *datastream.StreamConfig) { c.StreamID = "" }, datastream.ErrStreamIDRequired},
		{caseMissingDisplayName, func(c *datastream.StreamConfig) { c.DisplayName = "" }, datastream.ErrDisplayNameRequired},
		{caseMissingLocation, func(c *datastream.StreamConfig) { c.Location = "" }, datastream.ErrLocationRequired},
		{"missing publication", func(c *datastream.StreamConfig) { c.Publication = "" }, datastream.ErrPublicationRequired},
		{
			"missing replication slot",
			func(c *datastream.StreamConfig) { c.ReplicationSlot = "" },
			datastream.ErrReplicationSlotRequired,
		},
		{"missing dataset", func(c *datastream.StreamConfig) { c.DatasetID = "" }, datastream.ErrDatasetRequired},
		{"unknown state", func(c *datastream.StreamConfig) { c.DesiredState = "STARTED" }, datastream.ErrStateUnknown},
		{
			"nameless schema",
			func(c *datastream.StreamConfig) {
				c.IncludeSchemas = []datastream.IncludeSchema{{Tables: []string{"t"}}}
			},
			datastream.ErrSchemaRequired,
		},
		{
			"duplicate schema",
			func(c *datastream.StreamConfig) {
				c.IncludeSchemas = []datastream.IncludeSchema{{Schema: "public"}, {Schema: "public"}}
			},
			datastream.ErrDuplicateSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validStream()
			tt.mut(&c)

			if err := c.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestStreamValidateAcceptsEveryKnownState(t *testing.T) {
	t.Parallel()

	for _, state := range []string{"", datastream.StateRunning, datastream.StatePaused, datastream.StateNotStarted} {
		c := validStream()
		c.DesiredState = state

		if err := c.Validate(); err != nil {
			t.Errorf("DesiredState %q: unexpected error: %v", state, err)
		}
	}
}
