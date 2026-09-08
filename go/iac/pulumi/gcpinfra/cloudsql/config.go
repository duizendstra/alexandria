// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package cloudsql

import (
	"errors"
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/internal/names"
)

// User types accepted by the Cloud SQL Admin API. A built-in user
// authenticates with a password held in the instance; the two IAM types
// authenticate with a Google Cloud identity and have no password at all.
const (
	// UserTypeBuiltIn is a PostgreSQL user with a password.
	UserTypeBuiltIn = "BUILT_IN"
	// UserTypeIAMUser is a human Google Cloud identity.
	UserTypeIAMUser = "CLOUD_IAM_USER"
	// UserTypeIAMServiceAccount is a service account identity.
	UserTypeIAMServiceAccount = "CLOUD_IAM_SERVICE_ACCOUNT"
)

var (
	// ErrInstanceNameRequired means the instance has no name.
	ErrInstanceNameRequired = errors.New("cloudsql: instance Name is required")
	// ErrRegionRequired means the instance has no region.
	ErrRegionRequired = errors.New("cloudsql: instance Region is required")
	// ErrDatabaseVersionRequired means the instance has no engine version.
	ErrDatabaseVersionRequired = errors.New("cloudsql: instance DatabaseVersion is required")
	// ErrTierRequired means the instance has no machine type.
	ErrTierRequired = errors.New("cloudsql: instance Tier is required")
	// ErrDiskSizeInvalid means the instance disk size is not positive.
	ErrDiskSizeInvalid = errors.New("cloudsql: instance DiskSizeGB must be greater than zero")
	// ErrFlagNameRequired means a database flag has an empty name.
	ErrFlagNameRequired = errors.New("cloudsql: database flag Name is required")
	// ErrDuplicateFlagName means two database flags share a name.
	ErrDuplicateFlagName = errors.New("cloudsql: duplicate database flag Name")

	// ErrDatabaseNameRequired means a database has no name.
	ErrDatabaseNameRequired = errors.New("cloudsql: database Name is required")
	// ErrDuplicateDatabaseName means two databases share a name.
	ErrDuplicateDatabaseName = errors.New("cloudsql: duplicate database Name")

	// ErrUserNameRequired means a user has no name.
	ErrUserNameRequired = errors.New("cloudsql: user Name is required")
	// ErrUserTypeUnknown means a user carries a type the API does not accept.
	ErrUserTypeUnknown = errors.New("cloudsql: unknown user Type")
	// ErrUserPasswordRequired means a built-in user has no password.
	ErrUserPasswordRequired = errors.New("cloudsql: built-in user Password is required")
	// ErrUserPasswordNotAllowed means an IAM user carries a password.
	ErrUserPasswordNotAllowed = errors.New("cloudsql: IAM user must not carry a Password")
	// ErrDuplicateUserName means two users share a name.
	ErrDuplicateUserName = errors.New("cloudsql: duplicate user Name")
	// ErrPasswordType means a user password could not be marked as a Pulumi
	// secret output.
	ErrPasswordType = errors.New("cloudsql: user password is not a string output")
	// ErrServerCACertType means the server CA certificate could not be
	// projected to a string output.
	ErrServerCACertType = errors.New("cloudsql: server CA certificate is not a string output")
)

// Flag is a single database flag applied to the instance.
type Flag struct {
	// Name is the flag name (e.g. "max_connections").
	Name string `json:"name"`
	// Value is the flag value, always as a string — the API takes strings
	// even for numeric and boolean flags.
	Value string `json:"value"`
}

// AuthorizedNetwork is one CIDR range allowed to reach the public IP.
type AuthorizedNetwork struct {
	// Name labels the range in the console.
	Name string `json:"name"`
	// Value is the CIDR range (e.g. "203.0.113.0/24").
	Value string `json:"value"`
}

// InstanceConfig defines a Cloud SQL PostgreSQL server.
//
// The zero value of the boolean fields is the safe end of each choice for a
// permanent stack that no caller has thought about yet — except Ipv4Enabled,
// where a server nobody can reach is the safe default, so a caller that wants
// a public IP asks for one.
type InstanceConfig struct {
	// Name is the instance identifier and the Pulumi logical name.
	Name string `json:"name"`
	// Region is the instance region (e.g. "europe-west4").
	Region string `json:"region"`
	// DatabaseVersion is the engine version enum (e.g. "POSTGRES_18").
	DatabaseVersion string `json:"databaseVersion"`
	// Tier is the machine type (e.g. "db-custom-1-3840").
	Tier string `json:"tier"`
	// Edition is "ENTERPRISE" or "ENTERPRISE_PLUS". Empty leaves the API
	// default for the tier.
	Edition string `json:"edition,omitempty"`
	// AvailabilityType is "ZONAL" or "REGIONAL". Empty leaves the API default.
	AvailabilityType string `json:"availabilityType,omitempty"`
	// DiskSizeGB is the initial data disk size in GB.
	DiskSizeGB int `json:"diskSizeGb"`
	// DiskType is "PD_SSD" or "PD_HDD". Empty leaves the API default.
	DiskType string `json:"diskType,omitempty"`
	// DiskAutoresize lets the disk grow past DiskSizeGB. Growth is not
	// reversible, so DiskSizeGB is a floor and not a budget.
	DiskAutoresize bool `json:"diskAutoresize,omitempty"`
	// Flags are database flags set on the instance.
	Flags []Flag `json:"flags,omitempty"`
	// Ipv4Enabled gives the instance a public IP address.
	Ipv4Enabled bool `json:"ipv4Enabled,omitempty"`
	// AuthorizedNetworks are the CIDR ranges allowed to reach the public IP.
	// An empty list with Ipv4Enabled leaves the instance reachable only by
	// callers that authenticate through the Cloud SQL connector.
	AuthorizedNetworks []AuthorizedNetwork `json:"authorizedNetworks,omitempty"`
	// SSLMode governs what the server requires of a connection (e.g.
	// "ENCRYPTED_ONLY"). Empty leaves the API default.
	SSLMode string `json:"sslMode,omitempty"`
	// ServerCAMode selects the certificate authority that issues the server
	// certificate (e.g. "GOOGLE_MANAGED_INTERNAL_CA"). Empty leaves the API
	// default.
	ServerCAMode string `json:"serverCaMode,omitempty"`
	// BackupEnabled turns on automated backups.
	BackupEnabled bool `json:"backupEnabled,omitempty"`
	// BackupStartTime is the daily backup window start, "HH:MM" UTC.
	BackupStartTime string `json:"backupStartTime,omitempty"`
	// PointInTimeRecovery turns on write-ahead-log archiving. It requires
	// BackupEnabled.
	PointInTimeRecovery bool `json:"pointInTimeRecovery,omitempty"`
	// TransactionLogRetentionDays bounds the point-in-time recovery window.
	TransactionLogRetentionDays int `json:"transactionLogRetentionDays,omitempty"`
	// RetainedBackups is how many automated backups to keep.
	RetainedBackups int `json:"retainedBackups,omitempty"`
	// Labels are key-value pairs for resource organization.
	Labels map[string]string `json:"labels,omitempty"`
}

// Validate checks that the instance configuration is complete.
func (c InstanceConfig) Validate() error { //nolint:gocritic // hugeParam: by-value keeps Config a plain value across the module
	switch {
	case c.Name == "":
		return ErrInstanceNameRequired
	case c.Region == "":
		return ErrRegionRequired
	case c.DatabaseVersion == "":
		return ErrDatabaseVersionRequired
	case c.Tier == "":
		return ErrTierRequired
	case c.DiskSizeGB <= 0:
		return ErrDiskSizeInvalid
	}

	for _, f := range c.Flags {
		if f.Name == "" {
			return ErrFlagNameRequired
		}
	}

	if name, dup := names.Duplicate(c.Flags, func(f *Flag) string { return f.Name }); dup {
		return fmt.Errorf("%w %q", ErrDuplicateFlagName, name)
	}

	return nil
}

// DatabaseConfig defines one database on the instance.
type DatabaseConfig struct {
	// Name is the database name and the Pulumi logical name.
	Name string `json:"name"`
	// Charset is the database character set. Empty leaves the API default.
	Charset string `json:"charset,omitempty"`
	// Collation is the database collation. Empty leaves the API default.
	Collation string `json:"collation,omitempty"`
}

// Validate checks that the database configuration is complete.
func (c DatabaseConfig) Validate() error {
	if c.Name == "" {
		return ErrDatabaseNameRequired
	}

	return nil
}

// ValidateDatabases checks a slice of databases for completeness and for
// logical-name uniqueness within the slice.
func ValidateDatabases(dbs []DatabaseConfig) error {
	for _, d := range dbs {
		if err := d.Validate(); err != nil {
			return err
		}
	}

	if name, dup := names.Duplicate(dbs, func(d *DatabaseConfig) string { return d.Name }); dup {
		return fmt.Errorf("%w %q", ErrDuplicateDatabaseName, name)
	}

	return nil
}

// UserConfig defines one user on the instance.
//
// An IAM user's Name is the identity it authenticates as: the full address
// for a human, and the service account address without the ".gserviceaccount.com"
// suffix for a service account, which is what the Cloud SQL Admin API accepts.
// This package does not rewrite the name — a caller that passes the wrong
// shape gets the API's own error, not a silently different user.
type UserConfig struct {
	// Name is the user name and the Pulumi logical name.
	Name string `json:"name"`
	// Type is one of UserTypeBuiltIn, UserTypeIAMUser or
	// UserTypeIAMServiceAccount. Empty means UserTypeBuiltIn, matching the
	// API default.
	Type string `json:"type,omitempty"`
	// Password is the built-in user's password, sourced at deploy time by the
	// caller. It must be empty for the IAM types, which have no password.
	Password string `json:"-"`
}

// Validate checks that the user configuration is complete and internally
// consistent.
func (c UserConfig) Validate() error {
	if c.Name == "" {
		return ErrUserNameRequired
	}

	switch c.Type {
	case "", UserTypeBuiltIn:
		if c.Password == "" {
			return fmt.Errorf("%w: %q", ErrUserPasswordRequired, c.Name)
		}
	case UserTypeIAMUser, UserTypeIAMServiceAccount:
		if c.Password != "" {
			return fmt.Errorf("%w: %q", ErrUserPasswordNotAllowed, c.Name)
		}
	default:
		return fmt.Errorf("%w %q for user %q", ErrUserTypeUnknown, c.Type, c.Name)
	}

	return nil
}

// ValidateUsers checks a slice of users for completeness and for
// logical-name uniqueness within the slice.
func ValidateUsers(users []UserConfig) error {
	for _, u := range users {
		if err := u.Validate(); err != nil {
			return err
		}
	}

	if name, dup := names.Duplicate(users, func(u *UserConfig) string { return u.Name }); dup {
		return fmt.Errorf("%w %q", ErrDuplicateUserName, name)
	}

	return nil
}
