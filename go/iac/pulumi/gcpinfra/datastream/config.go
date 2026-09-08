// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datastream

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/internal/names"
)

// Stream states accepted by the Datastream API.
const (
	// StateRunning replicates continuously.
	StateRunning = "RUNNING"
	// StatePaused keeps the stream and its position but stops replicating.
	StatePaused = "PAUSED"
	// StateNotStarted leaves a created stream idle until it is first started.
	StateNotStarted = "NOT_STARTED"
)

// DefaultPostgresPort is the port a PostgreSQL profile uses when the caller
// names none.
const DefaultPostgresPort = 5432

var (
	// ErrProfileIDRequired means a connection profile has no identifier.
	ErrProfileIDRequired = errors.New("datastream: profile ID is required")
	// ErrDisplayNameRequired means a resource has no human-readable name.
	ErrDisplayNameRequired = errors.New("datastream: DisplayName is required")
	// ErrLocationRequired means a resource has no region.
	ErrLocationRequired = errors.New("datastream: Location is required")
	// ErrHostnameRequired means a PostgreSQL profile has no host.
	ErrHostnameRequired = errors.New("datastream: profile Hostname is required")
	// ErrUsernameRequired means a PostgreSQL profile has no user.
	ErrUsernameRequired = errors.New("datastream: profile Username is required")
	// ErrDatabaseRequired means a PostgreSQL profile names no database.
	ErrDatabaseRequired = errors.New("datastream: profile Database is required")
	// ErrCredentialRequired means a PostgreSQL profile carries neither a
	// password nor a Secret Manager reference to one.
	ErrCredentialRequired = errors.New("datastream: profile needs either Password or SecretManagerStoredPassword")
	// ErrCredentialAmbiguous means a PostgreSQL profile carries both.
	ErrCredentialAmbiguous = errors.New("datastream: profile carries both Password and SecretManagerStoredPassword")
	// ErrPortInvalid means a PostgreSQL profile names an impossible port.
	ErrPortInvalid = errors.New("datastream: profile Port is out of range")

	// ErrStreamIDRequired means a stream has no identifier.
	ErrStreamIDRequired = errors.New("datastream: StreamID is required")
	// ErrPublicationRequired means a stream names no PostgreSQL publication.
	ErrPublicationRequired = errors.New("datastream: Publication is required")
	// ErrReplicationSlotRequired means a stream names no replication slot.
	ErrReplicationSlotRequired = errors.New("datastream: ReplicationSlot is required")
	// ErrDatasetRequired means a stream names no destination dataset.
	ErrDatasetRequired = errors.New("datastream: DatasetID is required")
	// ErrStateUnknown means a stream carries a state the API does not accept.
	ErrStateUnknown = errors.New("datastream: unknown DesiredState")
	// ErrSchemaRequired means an include-list entry names no schema.
	ErrSchemaRequired = errors.New("datastream: include Schema is required")
	// ErrDuplicateSchema means two include-list entries name one schema.
	ErrDuplicateSchema = errors.New("datastream: duplicate include Schema")
	// ErrPasswordType means a profile password could not be marked as a
	// Pulumi secret output.
	ErrPasswordType = errors.New("datastream: profile password is not a string output")
)

// PostgresProfileConfig defines how Datastream reaches a PostgreSQL source.
//
// The credential is either a literal Password, which lands encrypted in the
// Pulumi state, or SecretManagerStoredPassword, the resource name of a secret
// version that Datastream reads for itself. Exactly one of the two is set: a
// profile with both is a configuration whose effective credential nobody can
// name by reading it.
type PostgresProfileConfig struct {
	// ID is the connection profile identifier and the Pulumi logical name.
	ID string `json:"id"`
	// DisplayName is the human-readable name.
	DisplayName string `json:"displayName"`
	// Location is the region (e.g. "europe-west4").
	Location string `json:"location"`
	// Hostname is the address Datastream connects to.
	Hostname string `json:"hostname"`
	// Port is the PostgreSQL port. Zero means DefaultPostgresPort.
	Port int `json:"port,omitempty"`
	// Username is the replication user.
	Username string `json:"username"`
	// Database is the database the stream reads from.
	Database string `json:"database"`
	// Password is the replication user's password, sourced at deploy time by
	// the caller.
	Password string `json:"-"`
	// SecretManagerStoredPassword is the full resource name of a secret
	// version holding the password (projects/…/secrets/…/versions/…).
	SecretManagerStoredPassword string `json:"secretManagerStoredPassword,omitempty"`
	// ServerCACertificate turns on server-only TLS verification: Datastream
	// checks the server against this PEM and presents no client certificate.
	// Empty leaves the API default.
	ServerCACertificate string `json:"-"`
	// Labels are key-value pairs for resource organization.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the source profile configuration is complete.
func (c PostgresProfileConfig) Validate() error { //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	switch {
	case c.ID == "":
		return ErrProfileIDRequired
	case c.DisplayName == "":
		return ErrDisplayNameRequired
	case c.Location == "":
		return ErrLocationRequired
	case c.Hostname == "":
		return ErrHostnameRequired
	case c.Username == "":
		return ErrUsernameRequired
	case c.Database == "":
		return ErrDatabaseRequired
	case c.Port < 0 || c.Port > 65535:
		return fmt.Errorf("%w: %d", ErrPortInvalid, c.Port)
	}

	hasPassword := c.Password != ""
	hasSecret := c.SecretManagerStoredPassword != ""

	switch {
	case hasPassword && hasSecret:
		return fmt.Errorf("%w: %q", ErrCredentialAmbiguous, c.ID)
	case !hasPassword && !hasSecret:
		return fmt.Errorf("%w: %q", ErrCredentialRequired, c.ID)
	}

	return nil
}

// BigQueryProfileConfig defines a BigQuery destination.
//
// It carries no BigQuery coordinates: the API's BigQuery profile is an empty
// object, and the dataset a stream writes to is named on the stream, not here.
type BigQueryProfileConfig struct {
	// ID is the connection profile identifier and the Pulumi logical name.
	ID string `json:"id"`
	// DisplayName is the human-readable name.
	DisplayName string `json:"displayName"`
	// Location is the region (e.g. "europe-west4").
	Location string `json:"location"`
	// Labels are key-value pairs for resource organization.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the destination profile configuration is complete.
func (c BigQueryProfileConfig) Validate() error {
	switch {
	case c.ID == "":
		return ErrProfileIDRequired
	case c.DisplayName == "":
		return ErrDisplayNameRequired
	case c.Location == "":
		return ErrLocationRequired
	}

	return nil
}

// IncludeSchema narrows a stream to named schemas, and optionally to named
// tables within them. An entry with no tables includes the whole schema,
// including tables created later.
type IncludeSchema struct {
	// Schema is the PostgreSQL schema name.
	Schema string `json:"schema"`
	// Tables are the tables to replicate. Empty means all of them.
	Tables []string `json:"tables,omitempty"`
}

// StreamConfig defines a PostgreSQL-to-BigQuery stream. The two connection
// profiles it joins are passed to ApplyStream as arguments rather than held
// here, because they are references to resources and not configuration a stack
// file can carry.
type StreamConfig struct {
	// StreamID is the stream identifier and the Pulumi logical name.
	StreamID string `json:"streamId"`
	// DisplayName is the human-readable name.
	DisplayName string `json:"displayName"`
	// Location is the region (e.g. "europe-west4").
	Location string `json:"location"`
	// Publication is the PostgreSQL publication the stream reads. It must
	// already exist in the source database.
	Publication string `json:"publication"`
	// ReplicationSlot is the PostgreSQL logical replication slot the stream
	// consumes. It must already exist in the source database.
	ReplicationSlot string `json:"replicationSlot"`
	// IncludeSchemas narrows what is replicated. Empty replicates everything
	// the publication carries.
	IncludeSchemas []IncludeSchema `json:"includeSchemas,omitempty"`
	// DatasetID is the single BigQuery dataset every source table lands in.
	DatasetID string `json:"datasetId"`
	// DataFreshness is how stale the destination may be, as a duration with a
	// trailing "s" (e.g. "0s" writes on every change). Empty leaves the API
	// default, which is not zero.
	DataFreshness string `json:"dataFreshness,omitempty"`
	// DesiredState is StateRunning, StatePaused or StateNotStarted. Empty
	// leaves the API default.
	DesiredState string `json:"desiredState,omitempty"`
	// BackfillAll replicates the existing rows once before following the log.
	// Without it the stream carries only changes made after it started.
	BackfillAll bool `json:"backfillAll,omitempty"`
	// Labels are key-value pairs for resource organization.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the stream configuration is complete.
func (c StreamConfig) Validate() error { //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	switch {
	case c.StreamID == "":
		return ErrStreamIDRequired
	case c.DisplayName == "":
		return ErrDisplayNameRequired
	case c.Location == "":
		return ErrLocationRequired
	case c.Publication == "":
		return ErrPublicationRequired
	case c.ReplicationSlot == "":
		return ErrReplicationSlotRequired
	case c.DatasetID == "":
		return ErrDatasetRequired
	}

	switch c.DesiredState {
	case "", StateRunning, StatePaused, StateNotStarted:
	default:
		return fmt.Errorf("%w %q", ErrStateUnknown, c.DesiredState)
	}

	for _, s := range c.IncludeSchemas {
		if s.Schema == "" {
			return ErrSchemaRequired
		}
	}

	if name, dup := names.Duplicate(c.IncludeSchemas, func(s *IncludeSchema) string { return s.Schema }); dup {
		return fmt.Errorf("%w %q", ErrDuplicateSchema, name)
	}

	return nil
}
