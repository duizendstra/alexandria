// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datastream

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/datastream"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ProfileOutputs holds references to a created connection profile.
type ProfileOutputs struct {
	// ID is the profile's full resource name, which is what a stream names.
	ID pulumi.IDOutput
	// Name is the profile name as the API assigned it.
	Name pulumi.StringOutput
	// Profile is the resource itself, for callers that need to depend on it.
	Profile *datastream.ConnectionProfile
}

// StreamOutputs holds references to a created stream.
type StreamOutputs struct {
	// Name is the stream's full resource name.
	Name pulumi.StringOutput
	// State is the state the API reports, which lags DesiredState while a
	// stream is starting or draining.
	State pulumi.StringOutput
	// Stream is the resource itself, for callers that need to depend on it.
	Stream *datastream.Stream
}

// ApplyPostgresProfile creates a connection profile pointing at a PostgreSQL
// server.
//
// Connection profiles are left unprotected: they hold no data, and replacing
// one costs nothing beyond the streams that reference it, which Pulumi
// sequences on their own. The credential inside is a different matter — a
// literal Password is marked as a Pulumi secret, so it is encrypted in the
// state file rather than absent from it.
func ApplyPostgresProfile(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg PostgresProfileConfig, //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	deps []pulumi.Resource,
	_ ...lifecycle.Option,
) (*ProfileOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	port := cfg.Port
	if port == 0 {
		port = DefaultPostgresPort
	}

	profile := &datastream.ConnectionProfilePostgresqlProfileArgs{
		Hostname: pulumi.String(cfg.Hostname),
		Port:     pulumi.Int(port),
		Username: pulumi.String(cfg.Username),
		Database: pulumi.String(cfg.Database),
	}

	if cfg.Password != "" {
		password, ok := pulumi.ToSecret(pulumi.String(cfg.Password)).(pulumi.StringOutput)
		if !ok {
			return nil, fmt.Errorf("%w %q", ErrPasswordType, cfg.ID)
		}

		profile.Password = password
	}

	if cfg.SecretManagerStoredPassword != "" {
		profile.SecretManagerStoredPassword = pulumi.String(cfg.SecretManagerStoredPassword)
	}

	if cfg.ServerCACertificate != "" {
		profile.SslConfig = &datastream.ConnectionProfilePostgresqlProfileSslConfigArgs{
			ServerVerification: &datastream.ConnectionProfilePostgresqlProfileSslConfigServerVerificationArgs{
				CaCertificate: pulumi.String(cfg.ServerCACertificate),
			},
		}
	}

	cp, err := datastream.NewConnectionProfile(ctx, cfg.ID, &datastream.ConnectionProfileArgs{
		Project:             projectID,
		ConnectionProfileId: pulumi.String(cfg.ID),
		DisplayName:         pulumi.String(cfg.DisplayName),
		Location:            pulumi.String(cfg.Location),
		Labels:              stringMap(cfg.Labels),
		PostgresqlProfile:   profile,
	}, pulumi.DependsOn(deps), pulumi.Protect(false))
	if err != nil {
		return nil, fmt.Errorf("create postgres connection profile %s: %w", cfg.ID, err)
	}

	return &ProfileOutputs{ID: cp.ID(), Name: cp.Name, Profile: cp}, nil
}

// ApplyBigQueryProfile creates a connection profile pointing at BigQuery.
//
// Like the source profile it is left unprotected: it holds nothing, not even
// the dataset, which is named on the stream.
func ApplyBigQueryProfile(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg BigQueryProfileConfig,
	deps []pulumi.Resource,
	_ ...lifecycle.Option,
) (*ProfileOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cp, err := datastream.NewConnectionProfile(ctx, cfg.ID, &datastream.ConnectionProfileArgs{
		Project:             projectID,
		ConnectionProfileId: pulumi.String(cfg.ID),
		DisplayName:         pulumi.String(cfg.DisplayName),
		Location:            pulumi.String(cfg.Location),
		Labels:              stringMap(cfg.Labels),
		BigqueryProfile:     &datastream.ConnectionProfileBigqueryProfileArgs{},
	}, pulumi.DependsOn(deps), pulumi.Protect(false))
	if err != nil {
		return nil, fmt.Errorf("create bigquery connection profile %s: %w", cfg.ID, err)
	}

	return &ProfileOutputs{ID: cp.ID(), Name: cp.Name, Profile: cp}, nil
}

// ApplyStream creates a stream joining a PostgreSQL source profile to a
// BigQuery destination profile.
//
// The stream is protected unless the caller passes lifecycle.Ephemeral. It
// holds no rows, but it does hold a position in the source's write-ahead log,
// and that position cannot be recovered: a replacement stream either backfills
// the whole source again or silently starts from the moment it was created,
// leaving a gap nothing downstream reports.
//
// cfg.Publication and cfg.ReplicationSlot must already exist in the source
// database. Pulumi cannot create them and will not notice they are missing —
// the Datastream API validates them when the stream is created, and again when
// it starts.
func ApplyStream(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	sourceProfile, destinationProfile pulumi.StringInput,
	cfg StreamConfig, //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	deps []pulumi.Resource,
	opts ...lifecycle.Option,
) (*StreamOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	source := &datastream.StreamSourceConfigPostgresqlSourceConfigArgs{
		Publication:     pulumi.String(cfg.Publication),
		ReplicationSlot: pulumi.String(cfg.ReplicationSlot),
	}

	if len(cfg.IncludeSchemas) > 0 {
		source.IncludeObjects = includeObjects(cfg.IncludeSchemas)
	}

	destination := &datastream.StreamDestinationConfigBigqueryDestinationConfigArgs{
		SingleTargetDataset: &datastream.StreamDestinationConfigBigqueryDestinationConfigSingleTargetDatasetArgs{
			DatasetId: pulumi.String(cfg.DatasetID),
		},
	}

	if cfg.DataFreshness != "" {
		destination.DataFreshness = pulumi.String(cfg.DataFreshness)
	}

	args := &datastream.StreamArgs{
		Project:     projectID,
		StreamId:    pulumi.String(cfg.StreamID),
		DisplayName: pulumi.String(cfg.DisplayName),
		Location:    pulumi.String(cfg.Location),
		Labels:      stringMap(cfg.Labels),
		SourceConfig: &datastream.StreamSourceConfigArgs{
			SourceConnectionProfile: sourceProfile,
			PostgresqlSourceConfig:  source,
		},
		DestinationConfig: &datastream.StreamDestinationConfigArgs{
			DestinationConnectionProfile: destinationProfile,
			BigqueryDestinationConfig:    destination,
		},
	}

	if cfg.DesiredState != "" {
		args.DesiredState = pulumi.String(cfg.DesiredState)
	}

	// Exactly one of the two backfill blocks must be present: the API has no
	// "unset", and leaving both out is rejected.
	if cfg.BackfillAll {
		args.BackfillAll = &datastream.StreamBackfillAllArgs{}
	} else {
		args.BackfillNone = &datastream.StreamBackfillNoneArgs{}
	}

	st, err := datastream.NewStream(ctx, cfg.StreamID, args, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
	if err != nil {
		return nil, fmt.Errorf("create stream %s: %w", cfg.StreamID, err)
	}

	return &StreamOutputs{Name: st.Name, State: st.State, Stream: st}, nil
}

// includeObjects maps the flat include list onto the provider's nested schema
// and table blocks.
func includeObjects(schemas []IncludeSchema) *datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsArgs {
	out := make(datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsPostgresqlSchemaArray, 0, len(schemas))

	for _, s := range schemas {
		tables := make(datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsPostgresqlSchemaPostgresqlTableArray, 0, len(s.Tables))
		for _, t := range s.Tables {
			tables = append(tables, &datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsPostgresqlSchemaPostgresqlTableArgs{
				Table: pulumi.String(t),
			})
		}

		entry := &datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsPostgresqlSchemaArgs{
			Schema: pulumi.String(s.Schema),
		}
		if len(tables) > 0 {
			entry.PostgresqlTables = tables
		}

		out = append(out, entry)
	}

	return &datastream.StreamSourceConfigPostgresqlSourceConfigIncludeObjectsArgs{PostgresqlSchemas: out}
}

func stringMap(m map[string]string) pulumi.StringMap {
	out := make(pulumi.StringMap, len(m))
	for k, v := range m {
		out[k] = pulumi.String(v)
	}

	return out
}
