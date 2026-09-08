// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datastream_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datastream"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// sent keeps every resource's inputs under its type token and name, so a test
// asserts on what the provider would have received.
type sent struct {
	mu      sync.Mutex
	seen    map[string]resource.PropertyMap
	protect map[string]bool
}

func (s *sent) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	key := args.TypeToken + "::" + args.Name

	s.mu.Lock()
	if s.seen == nil {
		s.seen = map[string]resource.PropertyMap{}
		s.protect = map[string]bool{}
	}

	s.seen[key] = args.Inputs.Copy()
	s.protect[key] = args.RegisterRPC.GetProtect()
	s.mu.Unlock()

	outputs := args.Inputs.Copy()
	outputs["name"] = resource.NewStringProperty(args.Name)

	return args.Name + "-id", outputs, nil
}

func (s *sent) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

const (
	tokenProfile = "gcp:datastream/connectionProfile:ConnectionProfile"
	tokenStream  = "gcp:datastream/stream:Stream"

	project     = "example-project"
	stack       = "stack"
	sourceID    = "source"
	streamID    = "app-cdc"
	pgProfile   = "postgresqlProfile"
	schemaName  = "public"
	tableOrders = "orders"
	labelEnv    = "env"
	labelValue  = "test"
)

// apply runs body against the mock monitor and returns what it registered.
func apply(t *testing.T, body func(ctx *pulumi.Context) error) *sent {
	t.Helper()

	rec := &sent{}
	if err := pulumi.RunErr(body, pulumi.WithMocks(project, stack, rec)); err != nil {
		t.Fatalf("pulumi run: %v", err)
	}

	return rec
}

func (s *sent) inputs(t *testing.T, key string) resource.PropertyMap {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	got, ok := s.seen[key]
	if !ok {
		t.Fatalf("nothing registered under %q", key)
	}

	return got
}

func (s *sent) protected(t *testing.T, key string) bool {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	got, ok := s.protect[key]
	if !ok {
		t.Fatalf("nothing registered under %q", key)
	}

	return got
}

// plain strips the secret and output envelopes the engine wraps values in.
func plain(v resource.PropertyValue) resource.PropertyValue {
	for {
		switch {
		case v.IsSecret():
			v = v.SecretValue().Element
		case v.IsOutput():
			v = v.OutputValue().Element
		default:
			return v
		}
	}
}

func at(t *testing.T, m resource.PropertyMap, key string) resource.PropertyValue {
	t.Helper()

	v, ok := m[resource.PropertyKey(key)]
	if !ok {
		t.Fatalf("property %q not sent; sent: %v", key, m.Mappable())
	}

	return plain(v)
}

func nested(t *testing.T, m resource.PropertyMap, keys ...string) resource.PropertyMap {
	t.Helper()

	for _, k := range keys {
		v := at(t, m, k)
		if !v.IsObject() {
			t.Fatalf("property %q is %v, want an object", k, v)
		}

		m = v.ObjectValue()
	}

	return m
}

func text(t *testing.T, m resource.PropertyMap, key string) string {
	t.Helper()

	v := at(t, m, key)
	if !v.IsString() {
		t.Fatalf("property %q is %v, want a string", key, v)
	}

	return v.StringValue()
}

func projectID() pulumi.StringOutput {
	return pulumi.String(project).ToStringOutput()
}

func profiles() (source, destination pulumi.StringInput) {
	return pulumi.String("projects/p/locations/l/connectionProfiles/source"),
		pulumi.String("projects/p/locations/l/connectionProfiles/sink")
}

// TestApplyPostgresProfileDefaultsThePort pins the one value this package
// supplies that the caller did not: a config with no Port must still reach the
// API with PostgreSQL's port, not with a zero the API rejects.
func TestApplyPostgresProfileDefaultsThePort(t *testing.T) {
	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyPostgresProfile(ctx, projectID(), validSource(), nil)

		return err
	})

	pg := nested(t, rec.inputs(t, tokenProfile+"::"+sourceID), pgProfile)
	if got := at(t, pg, "port"); !got.IsNumber() || got.NumberValue() != datastream.DefaultPostgresPort {
		t.Errorf("port = %v, want %d", got, datastream.DefaultPostgresPort)
	}
}

// TestApplyPostgresProfileSendsPasswordAsSecret: a literal password reaches
// the state file, and it must reach it encrypted.
func TestApplyPostgresProfileSendsPasswordAsSecret(t *testing.T) {
	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyPostgresProfile(ctx, projectID(), validSource(), nil)

		return err
	})

	pg := nested(t, rec.inputs(t, tokenProfile+"::"+sourceID), pgProfile)

	raw, ok := pg["password"]
	if !ok {
		t.Fatalf("no password sent; sent: %v", pg.Mappable())
	}

	if !raw.ContainsSecrets() {
		t.Errorf("password was sent unencrypted as %v", raw)
	}
}

// TestApplyPostgresProfileSecretManagerCredential is the credential form that
// keeps the password out of the state file entirely: a version name, plus the
// server certificate the profile verifies the server against.
func TestApplyPostgresProfileSecretManagerCredential(t *testing.T) {
	cfg := validSource()
	cfg.Password = ""
	cfg.Port = 6543
	cfg.SecretManagerStoredPassword = secretVersion
	cfg.ServerCACertificate = "-----BEGIN CERTIFICATE-----"
	cfg.Labels = map[string]string{labelEnv: labelValue}

	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyPostgresProfile(ctx, projectID(), cfg, nil)

		return err
	})

	inputs := rec.inputs(t, tokenProfile+"::"+sourceID)

	pg := nested(t, inputs, pgProfile)
	if got := text(t, pg, "secretManagerStoredPassword"); got != secretVersion {
		t.Errorf("secretManagerStoredPassword = %q, want %q", got, secretVersion)
	}

	if _, ok := pg["password"]; ok {
		t.Error("a Secret Manager credential must not also send a literal password")
	}

	if got := at(t, pg, "port"); !got.IsNumber() || got.NumberValue() != 6543 {
		t.Errorf("port = %v, want the configured 6543", got)
	}

	verification := nested(t, pg, "sslConfig", "serverVerification")
	if text(t, verification, "caCertificate") == "" {
		t.Error("server certificate was not sent")
	}
}

func TestApplyPostgresProfileRejectsInvalidConfig(t *testing.T) {
	apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyPostgresProfile(ctx, projectID(), datastream.PostgresProfileConfig{}, nil)
		if !errors.Is(err, datastream.ErrProfileIDRequired) {
			t.Errorf("expected ErrProfileIDRequired, got %v", err)
		}

		return nil
	})
}

func TestApplyBigQueryProfileCreates(t *testing.T) {
	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyBigQueryProfile(ctx, projectID(), datastream.BigQueryProfileConfig{
			ID:          sinkID,
			DisplayName: sinkDisplayName,
			Location:    region,
			Labels:      map[string]string{labelEnv: labelValue},
		}, nil)

		return err
	})

	inputs := rec.inputs(t, tokenProfile+"::"+sinkID)
	if got := text(t, inputs, "location"); got != region {
		t.Errorf("location = %q, want %q", got, region)
	}

	if _, ok := inputs["bigqueryProfile"]; !ok {
		t.Errorf("no bigqueryProfile block sent; sent: %v", inputs.Mappable())
	}
}

func TestApplyBigQueryProfileRejectsInvalidConfig(t *testing.T) {
	apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyBigQueryProfile(ctx, projectID(), datastream.BigQueryProfileConfig{ID: sinkID}, nil)
		if !errors.Is(err, datastream.ErrDisplayNameRequired) {
			t.Errorf("expected ErrDisplayNameRequired, got %v", err)
		}

		return nil
	})
}

// TestApplyStreamSendsBackfillNoneByDefault pins the comment in ApplyStream:
// the API has no "unset" for backfill, so leaving both blocks out is rejected.
// A stream that quietly gained a full backfill would re-read the whole source.
func TestApplyStreamSendsBackfillNoneByDefault(t *testing.T) {
	source, destination := profiles()

	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyStream(ctx, projectID(), source, destination, validStream(), nil)

		return err
	})

	inputs := rec.inputs(t, tokenStream+"::"+streamID)
	if _, ok := inputs["backfillNone"]; !ok {
		t.Errorf("no backfillNone block sent; sent: %v", inputs.Mappable())
	}

	if _, ok := inputs["backfillAll"]; ok {
		t.Error("backfillAll was sent for a stream configured not to backfill")
	}
}

func TestApplyStreamSendsBackfillAllWhenAsked(t *testing.T) {
	cfg := validStream()
	cfg.BackfillAll = true

	source, destination := profiles()

	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyStream(ctx, projectID(), source, destination, cfg, nil)

		return err
	})

	inputs := rec.inputs(t, tokenStream+"::"+streamID)
	if _, ok := inputs["backfillAll"]; !ok {
		t.Errorf("no backfillAll block sent; sent: %v", inputs.Mappable())
	}

	if _, ok := inputs["backfillNone"]; ok {
		t.Error("backfillNone was sent alongside backfillAll")
	}
}

// TestApplyStreamMapsIncludeObjects walks the two levels of nesting the
// provider wants for what the config states flatly.
func TestApplyStreamMapsIncludeObjects(t *testing.T) {
	cfg := validStream()
	cfg.IncludeSchemas = []datastream.IncludeSchema{
		{Schema: schemaName, Tables: []string{tableOrders}},
		{Schema: "audit"},
	}
	cfg.DataFreshness = "0s"
	cfg.DesiredState = datastream.StateRunning
	cfg.Labels = map[string]string{labelEnv: labelValue}

	source, destination := profiles()

	rec := apply(t, func(ctx *pulumi.Context) error {
		_, err := datastream.ApplyStream(ctx, projectID(), source, destination, cfg, nil)

		return err
	})

	inputs := rec.inputs(t, tokenStream+"::"+streamID)
	if got := text(t, inputs, "desiredState"); got != datastream.StateRunning {
		t.Errorf("desiredState = %q, want %q", got, datastream.StateRunning)
	}

	freshness := nested(t, inputs, "destinationConfig", "bigqueryDestinationConfig")
	if got := text(t, freshness, "dataFreshness"); got != "0s" {
		t.Errorf("dataFreshness = %q, want 0s", got)
	}

	target := nested(t, freshness, "singleTargetDataset")
	if got := text(t, target, "datasetId"); got != validStream().DatasetID {
		t.Errorf("datasetId = %q, want %q", got, validStream().DatasetID)
	}

	objects := nested(t, inputs, "sourceConfig", "postgresqlSourceConfig", "includeObjects")

	schemas := at(t, objects, "postgresqlSchemas")
	if !schemas.IsArray() || len(schemas.ArrayValue()) != 2 {
		t.Fatalf("postgresqlSchemas = %v, want two schemas", schemas)
	}

	first := plain(schemas.ArrayValue()[0]).ObjectValue()
	if got := text(t, first, "schema"); got != schemaName {
		t.Errorf("schema = %q, want %q", got, schemaName)
	}

	tables := at(t, first, "postgresqlTables")
	if !tables.IsArray() || len(tables.ArrayValue()) != 1 {
		t.Fatalf("postgresqlTables = %v, want one table", tables)
	}

	// A schema with no tables must send no table block at all, which is how
	// the API reads "every table in this schema".
	second := plain(schemas.ArrayValue()[1]).ObjectValue()
	if v, ok := second["postgresqlTables"]; ok && !plain(v).IsNull() {
		t.Errorf("a schema with no tables sent %v", v)
	}
}

// TestApplyStreamProtection pins what the doc comment claims: a stream holds
// a position in the source write-ahead log that a replacement cannot recover,
// so it is protected unless the caller says the whole stack is throwaway.
func TestApplyStreamProtection(t *testing.T) {
	source, destination := profiles()
	key := tokenStream + "::" + streamID

	tests := []struct {
		name string
		opts []lifecycle.Option
		want bool
	}{
		{name: "durable by default", want: true},
		{name: "ephemeral clears it", opts: []lifecycle.Option{lifecycle.Ephemeral()}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := apply(t, func(ctx *pulumi.Context) error {
				out, err := datastream.ApplyStream(ctx, projectID(), source, destination,
					validStream(), nil, tt.opts...)
				if err != nil {
					return err
				}

				if out.Stream == nil {
					t.Error("expected the stream resource in the outputs")
				}

				return nil
			})

			if got := rec.protected(t, key); got != tt.want {
				t.Errorf("stream protect = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestApplyProfilesAreNeverProtected is the matching negative. Both profiles
// are applied with the default options — the ones that protect a stream — and
// must still come out unprotected: a profile holds nothing, and protecting it
// would block the replacement Pulumi has to make when a hostname changes.
func TestApplyProfilesAreNeverProtected(t *testing.T) {
	rec := apply(t, func(ctx *pulumi.Context) error {
		if _, err := datastream.ApplyPostgresProfile(ctx, projectID(), validSource(), nil); err != nil {
			return err
		}

		_, err := datastream.ApplyBigQueryProfile(ctx, projectID(), datastream.BigQueryProfileConfig{
			ID:          sinkID,
			DisplayName: sinkDisplayName,
			Location:    region,
		}, nil)

		return err
	})

	for _, id := range []string{sourceID, sinkID} {
		if rec.protected(t, tokenProfile+"::"+id) {
			t.Errorf("connection profile %q is protected", id)
		}
	}
}

func TestApplyStreamRejectsInvalidConfig(t *testing.T) {
	source, destination := profiles()

	apply(t, func(ctx *pulumi.Context) error {
		cfg := validStream()
		cfg.Publication = ""

		_, err := datastream.ApplyStream(ctx, projectID(), source, destination, cfg, nil)
		if !errors.Is(err, datastream.ErrPublicationRequired) {
			t.Errorf("expected ErrPublicationRequired, got %v", err)
		}

		return nil
	})
}
