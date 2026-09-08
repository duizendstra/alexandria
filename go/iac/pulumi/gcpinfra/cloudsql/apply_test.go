// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package cloudsql_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudsql"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// recorder keeps every resource's inputs so a test can assert on what the
// provider would have been sent, rather than only on the absence of an error.
type recorder struct {
	mu     sync.Mutex
	inputs map[string]resource.PropertyMap
}

func newRecorder() *recorder {
	return &recorder{inputs: map[string]resource.PropertyMap{}}
}

func (r *recorder) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	r.mu.Lock()
	r.inputs[args.TypeToken+"::"+args.Name] = args.Inputs.Copy()
	r.mu.Unlock()

	outputs := args.Inputs.Copy()
	outputs["name"] = resource.NewStringProperty(args.Name)

	return args.Name + "-id", outputs, nil
}

func (r *recorder) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (r *recorder) get(t *testing.T, key string) resource.PropertyMap {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	got, ok := r.inputs[key]
	if !ok {
		t.Fatalf("no resource registered under %q; registered: %v", key, r.keys())
	}

	return got
}

// keys is called with the lock already held, by get, on the failure path only.
func (r *recorder) keys() []string {
	out := make([]string, 0, len(r.inputs))
	for k := range r.inputs {
		out = append(out, k)
	}

	return out
}

const (
	tokenInstance = "gcp:sql/databaseInstance:DatabaseInstance"
	tokenDatabase = "gcp:sql/database:Database"
	tokenUser     = "gcp:sql/user:User"

	keySettings = "settings"
	keyBackup   = "backupConfiguration"

	projectID = "example-project"
	stackName = "stack"
	charsetU8 = "UTF8"
)

// unwrap strips the secret and output envelopes the engine puts around a value
// so an assertion reads the value itself.
func unwrap(v resource.PropertyValue) resource.PropertyValue {
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

func field(t *testing.T, m resource.PropertyMap, key string) resource.PropertyValue {
	t.Helper()

	v, ok := m[resource.PropertyKey(key)]
	if !ok {
		t.Fatalf("property %q not sent; sent: %v", key, m.Mappable())
	}

	return unwrap(v)
}

func objectAt(t *testing.T, m resource.PropertyMap, key string) resource.PropertyMap {
	t.Helper()

	v := field(t, m, key)
	if !v.IsObject() {
		t.Fatalf("property %q is %v, want an object", key, v)
	}

	return v.ObjectValue()
}

func boolAt(t *testing.T, m resource.PropertyMap, key string) bool {
	t.Helper()

	v := field(t, m, key)
	if !v.IsBool() {
		t.Fatalf("property %q is %v, want a bool", key, v)
	}

	return v.BoolValue()
}

func stringAt(t *testing.T, m resource.PropertyMap, key string) string {
	t.Helper()

	v := field(t, m, key)
	if !v.IsString() {
		t.Fatalf("property %q is %v, want a string", key, v)
	}

	return v.StringValue()
}

func fullInstance() cloudsql.InstanceConfig {
	return cloudsql.InstanceConfig{
		Name:                        instanceName,
		Region:                      region,
		DatabaseVersion:             version,
		Tier:                        tier,
		Edition:                     "ENTERPRISE",
		AvailabilityType:            "ZONAL",
		DiskSizeGB:                  10,
		DiskType:                    "PD_SSD",
		DiskAutoresize:              true,
		Flags:                       []cloudsql.Flag{{Name: flagLogicalDecoding, Value: "on"}},
		Ipv4Enabled:                 true,
		AuthorizedNetworks:          []cloudsql.AuthorizedNetwork{{Name: "office", Value: "203.0.113.0/24"}},
		SSLMode:                     "ENCRYPTED_ONLY",
		ServerCAMode:                "GOOGLE_MANAGED_INTERNAL_CA",
		BackupEnabled:               true,
		BackupStartTime:             "02:00",
		PointInTimeRecovery:         true,
		TransactionLogRetentionDays: 7,
		RetainedBackups:             14,
		Labels:                      map[string]string{"env": "test"},
	}
}

func runWith(t *testing.T, body func(ctx *pulumi.Context) error) *recorder {
	t.Helper()

	rec := newRecorder()
	if err := pulumi.RunErr(body, pulumi.WithMocks(projectID, stackName, rec)); err != nil {
		t.Fatalf("pulumi run: %v", err)
	}

	return rec
}

// TestApplyInstanceSetsBothDeletionGuards pins the doc comment's claim that
// protection is set twice: once at the API and once in the engine's own
// settings block. A single guard would leave the instance deletable by the
// path the other one covers.
func TestApplyInstanceSetsBothDeletionGuards(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		_, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(), fullInstance(), nil)

		return err
	})

	inputs := rec.get(t, tokenInstance+"::"+instanceName)
	if !boolAt(t, inputs, "deletionProtection") {
		t.Error("API-level deletionProtection is false on a durable instance")
	}

	settings := objectAt(t, inputs, keySettings)
	if !boolAt(t, settings, "deletionProtectionEnabled") {
		t.Error("settings-level deletionProtectionEnabled is false on a durable instance")
	}
}

// TestApplyInstanceEphemeralClearsBothGuards is the other half: an ephemeral
// stack must be destroyable, and a guard left standing at either level blocks
// the destroy with an error nothing in the stack can clear.
func TestApplyInstanceEphemeralClearsBothGuards(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		_, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(),
			fullInstance(), nil, lifecycle.Ephemeral())

		return err
	})

	inputs := rec.get(t, tokenInstance+"::"+instanceName)
	if boolAt(t, inputs, "deletionProtection") {
		t.Error("API-level deletionProtection is true on an ephemeral instance")
	}

	settings := objectAt(t, inputs, keySettings)
	if boolAt(t, settings, "deletionProtectionEnabled") {
		t.Error("settings-level deletionProtectionEnabled is true on an ephemeral instance")
	}
}

// TestApplyInstanceMapsSettings walks the nested settings block, which is the
// part of this package a caller cannot see from the config type alone.
func TestApplyInstanceMapsSettings(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		_, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(), fullInstance(), nil)

		return err
	})

	inputs := rec.get(t, tokenInstance+"::"+instanceName)
	if got := stringAt(t, inputs, "databaseVersion"); got != version {
		t.Errorf("databaseVersion = %q, want %q", got, version)
	}

	settings := objectAt(t, inputs, keySettings)
	if got := stringAt(t, settings, "tier"); got != tier {
		t.Errorf("tier = %q, want %q", got, tier)
	}

	flags := field(t, settings, "databaseFlags")
	if !flags.IsArray() || len(flags.ArrayValue()) != 1 {
		t.Fatalf("databaseFlags = %v, want one flag", flags)
	}

	flag := unwrap(flags.ArrayValue()[0]).ObjectValue()
	if got := stringAt(t, flag, "name"); got != flagLogicalDecoding {
		t.Errorf("flag name = %q, want %q", got, flagLogicalDecoding)
	}

	backup := objectAt(t, settings, keyBackup)
	if got := stringAt(t, backup, "startTime"); got != "02:00" {
		t.Errorf("backup startTime = %q, want 02:00", got)
	}

	retention := objectAt(t, backup, "backupRetentionSettings")
	if got := field(t, retention, "retainedBackups"); !got.IsNumber() || got.NumberValue() != 14 {
		t.Errorf("retainedBackups = %v, want 14", got)
	}
}

// TestApplyInstanceOmitsUnsetOptionals pins the reason optionalString exists:
// an empty config field must send no field at all, so a preview on an
// untouched stack stays at zero changes instead of proposing an empty string.
func TestApplyInstanceOmitsUnsetOptionals(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		_, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(), cloudsql.InstanceConfig{
			Name:            instanceName,
			Region:          region,
			DatabaseVersion: version,
			Tier:            tier,
			DiskSizeGB:      10,
		}, nil)

		return err
	})

	settings := rec.get(t, tokenInstance+"::"+instanceName)
	settings = objectAt(t, settings, keySettings)

	for _, key := range []string{"edition", "availabilityType", "diskType"} {
		if v, ok := settings[resource.PropertyKey(key)]; ok && !unwrap(v).IsNull() {
			t.Errorf("unset %q was sent as %v", key, v)
		}
	}

	backup := objectAt(t, settings, keyBackup)
	if v, ok := backup["transactionLogRetentionDays"]; ok && !unwrap(v).IsNull() {
		t.Errorf("unset transactionLogRetentionDays was sent as %v", v)
	}
}

// TestApplyInstanceServerCACertificateBeforeCreate is a regression pin. The
// first version indexed ServerCaCerts[0] directly, which panics on the very
// preview that creates the instance, because the array is empty until the
// instance exists. The mock returns no certificates, reproducing that state.
func TestApplyInstanceServerCACertificateBeforeCreate(t *testing.T) {
	var (
		mu       sync.Mutex
		resolved bool
		gotCA    = "sentinel: the certificate output never resolved"
	)

	runWith(t, func(ctx *pulumi.Context) error {
		out, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(), fullInstance(), nil)
		if err != nil {
			return err
		}

		out.ServerCACertificate.ApplyT(func(ca string) string {
			mu.Lock()
			resolved, gotCA = true, ca
			mu.Unlock()

			return ca
		})

		return nil
	})

	mu.Lock()
	defer mu.Unlock()

	if !resolved {
		t.Fatal("ServerCACertificate never resolved, so this test proves nothing")
	}

	if gotCA != "" {
		t.Errorf("ServerCACertificate = %q, want empty before the instance exists", gotCA)
	}
}

func TestApplyInstanceRejectsInvalidConfig(t *testing.T) {
	runWith(t, func(ctx *pulumi.Context) error {
		_, err := cloudsql.ApplyInstance(ctx, pulumi.String(projectID).ToStringOutput(), cloudsql.InstanceConfig{}, nil)
		if !errors.Is(err, cloudsql.ErrInstanceNameRequired) {
			t.Errorf("expected ErrInstanceNameRequired, got %v", err)
		}

		return nil
	})
}

func TestApplyDatabasesCreatesAndOmitsUnsetOptionals(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		return cloudsql.ApplyDatabases(ctx, pulumi.String(projectID).ToStringOutput(),
			pulumi.String(instanceName).ToStringOutput(), []cloudsql.DatabaseConfig{
				{Name: dbApp, Charset: charsetU8, Collation: "en_US.UTF8"},
				{Name: "plain"},
			}, nil)
	})

	app := rec.get(t, tokenDatabase+"::"+dbApp)
	if got := stringAt(t, app, "charset"); got != charsetU8 {
		t.Errorf("charset = %q, want %q", got, charsetU8)
	}

	plain := rec.get(t, tokenDatabase+"::plain")
	if v, ok := plain["charset"]; ok && !unwrap(v).IsNull() {
		t.Errorf("unset charset was sent as %v", v)
	}
}

func TestApplyDatabasesRejectsDuplicates(t *testing.T) {
	runWith(t, func(ctx *pulumi.Context) error {
		err := cloudsql.ApplyDatabases(ctx, pulumi.String(projectID).ToStringOutput(),
			pulumi.String(instanceName).ToStringOutput(), []cloudsql.DatabaseConfig{
				{Name: dbApp}, {Name: dbApp},
			}, nil)
		if !errors.Is(err, cloudsql.ErrDuplicateDatabaseName) {
			t.Errorf("expected ErrDuplicateDatabaseName, got %v", err)
		}

		return nil
	})
}

// TestApplyUsersSendsTheBuiltInPassword pins that a built-in user's password
// reaches the provider at all, and reaches it encrypted.
//
// The secrecy half is a provider-contract pin, not a pin on this package: the
// pulumi-gcp schema marks sql.User's password field secret on its own, so
// deleting the pulumi.ToSecret call in ApplyUsers leaves this assertion green.
// The call stays because it is what keeps the guarantee inside this package if
// that schema ever changes — and this assertion is what would notice. What the
// test does catch on its own is the password being dropped or altered on the
// way through, which the value comparison below covers.
func TestApplyUsersSendsTheBuiltInPassword(t *testing.T) {
	const password = "correct horse battery staple"

	rec := runWith(t, func(ctx *pulumi.Context) error {
		return cloudsql.ApplyUsers(ctx, pulumi.String(projectID).ToStringOutput(),
			pulumi.String(instanceName).ToStringOutput(), []cloudsql.UserConfig{
				{Name: userSvc, Type: cloudsql.UserTypeBuiltIn, Password: password},
			}, nil)
	})

	inputs := rec.get(t, tokenUser+"::"+userSvc)

	raw, ok := inputs["password"]
	if !ok {
		t.Fatalf("no password sent; sent: %v", inputs.Mappable())
	}

	if !raw.ContainsSecrets() {
		t.Errorf("password was sent unencrypted as %v", raw)
	}

	if got := stringAt(t, inputs, "password"); got != password {
		t.Errorf("password = %q, want the configured value", got)
	}
}

// TestApplyUsersIAMUserCarriesNoPassword is the matching negative: an IAM user
// authenticates with a token, and a password field on it would be a credential
// the instance never checks.
func TestApplyUsersIAMUserCarriesNoPassword(t *testing.T) {
	rec := runWith(t, func(ctx *pulumi.Context) error {
		return cloudsql.ApplyUsers(ctx, pulumi.String(projectID).ToStringOutput(),
			pulumi.String(instanceName).ToStringOutput(), []cloudsql.UserConfig{
				{Name: userPerson, Type: cloudsql.UserTypeIAMUser},
			}, nil)
	})

	inputs := rec.get(t, tokenUser+"::"+userPerson)
	if got := stringAt(t, inputs, "type"); got != cloudsql.UserTypeIAMUser {
		t.Errorf("type = %q, want %q", got, cloudsql.UserTypeIAMUser)
	}

	if v, ok := inputs["password"]; ok && !unwrap(v).IsNull() {
		t.Errorf("IAM user was sent a password: %v", v)
	}
}

func TestApplyUsersRejectsInvalidConfig(t *testing.T) {
	runWith(t, func(ctx *pulumi.Context) error {
		err := cloudsql.ApplyUsers(ctx, pulumi.String(projectID).ToStringOutput(),
			pulumi.String(instanceName).ToStringOutput(), []cloudsql.UserConfig{
				{Name: userPerson, Type: cloudsql.UserTypeIAMUser, Password: "nope"},
			}, nil)
		if !errors.Is(err, cloudsql.ErrUserPasswordNotAllowed) {
			t.Errorf("expected ErrUserPasswordNotAllowed, got %v", err)
		}

		return nil
	})
}
