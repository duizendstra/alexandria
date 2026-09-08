// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package cloudsql

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// InstanceOutputs holds references to the created instance.
type InstanceOutputs struct {
	// Name is the instance name as the API assigned it.
	Name pulumi.StringOutput
	// ConnectionName is the "project:region:instance" triple the Cloud SQL
	// connector and the gcloud client take.
	ConnectionName pulumi.StringOutput
	// PublicIPAddress is empty unless the instance has a public IP.
	PublicIPAddress pulumi.StringOutput
	// ServerCACertificate is the PEM of the certificate authority that signed
	// the server certificate. A client that verifies the server — and only
	// the server — pins this.
	ServerCACertificate pulumi.StringOutput
	// Instance is the resource itself, for callers that need to depend on it.
	Instance *sql.DatabaseInstance
}

// ApplyInstance creates a Cloud SQL PostgreSQL server.
//
// cfg.Name is the Pulumi logical name as well as the instance name. Changing
// it replaces the server and every byte on it, so the instance is protected
// unless the caller passes lifecycle.Ephemeral. Protection is set twice on
// purpose: Pulumi's own protect refuses the delete in the engine, and the
// provider's deletion protection refuses it at the API, which also stops a
// delete issued outside this stack.
//
// Cloud SQL keeps an instance name reserved for about a week after a delete,
// so a destroyed ephemeral stack cannot immediately be recreated under the
// same name.
func ApplyInstance(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg InstanceConfig, //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	deps []pulumi.Resource,
	opts ...lifecycle.Option,
) (*InstanceOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ephemeral := lifecycle.IsEphemeral(opts...)

	labels := make(pulumi.StringMap, len(cfg.Labels))
	for k, v := range cfg.Labels {
		labels[k] = pulumi.String(v)
	}

	inst, err := sql.NewDatabaseInstance(ctx, cfg.Name, &sql.DatabaseInstanceArgs{
		Project:         projectID,
		Name:            pulumi.String(cfg.Name),
		Region:          pulumi.String(cfg.Region),
		DatabaseVersion: pulumi.String(cfg.DatabaseVersion),
		// The API-level guard, independent of the engine-level one below.
		DeletionProtection: pulumi.Bool(!ephemeral),
		Settings:           instanceSettings(cfg, ephemeral, labels),
	}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
	if err != nil {
		return nil, fmt.Errorf("create sql instance %s: %w", cfg.Name, err)
	}

	// Indexing the array output directly panics when it is empty, which it is
	// on the very preview that creates the instance. The length is checked
	// here instead, so a caller that pins this certificate gets an empty
	// string until the instance exists rather than a crashed run.
	serverCA, ok := inst.ServerCaCerts.ApplyT(
		func(certs []sql.DatabaseInstanceServerCaCert) string {
			if len(certs) == 0 || certs[0].Cert == nil {
				return ""
			}

			return *certs[0].Cert
		}).(pulumi.StringOutput)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServerCACertType, cfg.Name)
	}

	return &InstanceOutputs{
		Name:                inst.Name,
		ConnectionName:      inst.ConnectionName,
		PublicIPAddress:     inst.PublicIpAddress,
		ServerCACertificate: serverCA,
		Instance:            inst,
	}, nil
}

// instanceSettings maps the flat config onto the provider's nested settings
// block. Every optional string is left unset when empty so the API default
// stands and a `pulumi preview` on an untouched stack stays at zero changes.
func instanceSettings(
	cfg InstanceConfig, //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	ephemeral bool,
	labels pulumi.StringMap,
) *sql.DatabaseInstanceSettingsArgs {
	flags := make(sql.DatabaseInstanceSettingsDatabaseFlagArray, 0, len(cfg.Flags))
	for _, f := range cfg.Flags {
		flags = append(flags, &sql.DatabaseInstanceSettingsDatabaseFlagArgs{
			Name:  pulumi.String(f.Name),
			Value: pulumi.String(f.Value),
		})
	}

	networks := make(sql.DatabaseInstanceSettingsIpConfigurationAuthorizedNetworkArray, 0, len(cfg.AuthorizedNetworks))
	for _, n := range cfg.AuthorizedNetworks {
		networks = append(networks, &sql.DatabaseInstanceSettingsIpConfigurationAuthorizedNetworkArgs{
			Name:  pulumi.String(n.Name),
			Value: pulumi.String(n.Value),
		})
	}

	backup := &sql.DatabaseInstanceSettingsBackupConfigurationArgs{
		Enabled:                    pulumi.Bool(cfg.BackupEnabled),
		StartTime:                  optionalString(cfg.BackupStartTime),
		PointInTimeRecoveryEnabled: pulumi.Bool(cfg.PointInTimeRecovery),
	}

	if cfg.TransactionLogRetentionDays > 0 {
		backup.TransactionLogRetentionDays = pulumi.Int(cfg.TransactionLogRetentionDays)
	}

	if cfg.RetainedBackups > 0 {
		backup.BackupRetentionSettings = &sql.DatabaseInstanceSettingsBackupConfigurationBackupRetentionSettingsArgs{
			RetainedBackups: pulumi.Int(cfg.RetainedBackups),
		}
	}

	settings := &sql.DatabaseInstanceSettingsArgs{
		Tier:                      pulumi.String(cfg.Tier),
		DiskSize:                  pulumi.Int(cfg.DiskSizeGB),
		DiskAutoresize:            pulumi.Bool(cfg.DiskAutoresize),
		DeletionProtectionEnabled: pulumi.Bool(!ephemeral),
		DatabaseFlags:             flags,
		UserLabels:                labels,
		IpConfiguration: &sql.DatabaseInstanceSettingsIpConfigurationArgs{
			Ipv4Enabled:        pulumi.Bool(cfg.Ipv4Enabled),
			AuthorizedNetworks: networks,
			SslMode:            optionalString(cfg.SSLMode),
			ServerCaMode:       optionalString(cfg.ServerCAMode),
		},
		BackupConfiguration: backup,
		Edition:             optionalString(cfg.Edition),
		AvailabilityType:    optionalString(cfg.AvailabilityType),
		DiskType:            optionalString(cfg.DiskType),
	}

	return settings
}

// optionalString returns a nil input for an empty value, so the provider sends
// no field at all rather than an empty string the API would reject or record.
func optionalString(v string) pulumi.StringPtrInput { //nolint:ireturn // the provider's args fields are typed as this interface
	if v == "" {
		return nil
	}

	return pulumi.String(v)
}

// ApplyDatabases creates databases on an existing instance.
//
// A database's Name is its Pulumi logical name: changing it drops the database
// and creates an empty one, so each is protected unless the caller passes
// lifecycle.Ephemeral.
func ApplyDatabases(
	ctx *pulumi.Context,
	projectID, instanceName pulumi.StringOutput,
	dbs []DatabaseConfig,
	deps []pulumi.Resource,
	opts ...lifecycle.Option,
) error {
	if err := ValidateDatabases(dbs); err != nil {
		return err
	}

	for _, d := range dbs {
		_, err := sql.NewDatabase(ctx, d.Name, &sql.DatabaseArgs{
			Project:   projectID,
			Instance:  instanceName,
			Name:      pulumi.String(d.Name),
			Charset:   optionalString(d.Charset),
			Collation: optionalString(d.Collation),
		}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
		if err != nil {
			return fmt.Errorf("create database %s: %w", d.Name, err)
		}
	}

	return nil
}

// ApplyUsers creates users on an existing instance.
//
// Users are deliberately left unprotected: a user holds no rows, and removing
// one is how access is withdrawn. What a dropped user can take with it is the
// objects it owns, which is a database concern the API surfaces as an error
// rather than something this block can decide.
//
// A built-in user's password is marked as a Pulumi secret, so it is encrypted
// in the state file. It is still in the state file — an instance whose password
// must never be readable from the stack wants a user created outside it.
func ApplyUsers(
	ctx *pulumi.Context,
	projectID, instanceName pulumi.StringOutput,
	users []UserConfig,
	deps []pulumi.Resource,
	_ ...lifecycle.Option,
) error {
	if err := ValidateUsers(users); err != nil {
		return err
	}

	for _, u := range users {
		args := &sql.UserArgs{
			Project:  projectID,
			Instance: instanceName,
			Name:     pulumi.String(u.Name),
		}

		if u.Type != "" {
			args.Type = pulumi.String(u.Type)
		}

		if u.Password != "" {
			password, ok := pulumi.ToSecret(pulumi.String(u.Password)).(pulumi.StringOutput)
			if !ok {
				return fmt.Errorf("%w %q", ErrPasswordType, u.Name)
			}

			args.Password = password
		}

		if _, err := sql.NewUser(ctx, u.Name, args, pulumi.DependsOn(deps)); err != nil {
			return fmt.Errorf("create sql user %s: %w", u.Name, err)
		}
	}

	return nil
}
