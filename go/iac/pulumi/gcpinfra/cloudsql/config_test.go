// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package cloudsql_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudsql"
)

const (
	instanceName = "shared-postgres"
	region       = "europe-west4"
	version      = "POSTGRES_18"
	tier         = "db-custom-1-3840"

	flagLogicalDecoding = "cloudsql.logical_decoding"
	dbApp               = "app"
	userSvc             = "svc"
	userPerson          = "person@example.invalid"
)

func validInstance() cloudsql.InstanceConfig {
	return cloudsql.InstanceConfig{
		Name:            instanceName,
		Region:          region,
		DatabaseVersion: version,
		Tier:            tier,
		DiskSizeGB:      10,
	}
}

func TestInstanceValidateValid(t *testing.T) {
	t.Parallel()

	if err := validInstance().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstanceValidateRejectsIncomplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mut  func(*cloudsql.InstanceConfig)
		want error
	}{
		{"missing name", func(c *cloudsql.InstanceConfig) { c.Name = "" }, cloudsql.ErrInstanceNameRequired},
		{"missing region", func(c *cloudsql.InstanceConfig) { c.Region = "" }, cloudsql.ErrRegionRequired},
		{"missing version", func(c *cloudsql.InstanceConfig) { c.DatabaseVersion = "" }, cloudsql.ErrDatabaseVersionRequired},
		{"missing tier", func(c *cloudsql.InstanceConfig) { c.Tier = "" }, cloudsql.ErrTierRequired},
		{"zero disk", func(c *cloudsql.InstanceConfig) { c.DiskSizeGB = 0 }, cloudsql.ErrDiskSizeInvalid},
		{"negative disk", func(c *cloudsql.InstanceConfig) { c.DiskSizeGB = -1 }, cloudsql.ErrDiskSizeInvalid},
		{
			"nameless flag",
			func(c *cloudsql.InstanceConfig) { c.Flags = []cloudsql.Flag{{Value: "on"}} },
			cloudsql.ErrFlagNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validInstance()
			tt.mut(&c)

			if err := c.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

// A repeated flag name is not a duplicate URN — the flags are one field on one
// resource — but the API takes the last write silently, so the config that set
// a flag twice is a config whose effective value nobody can read off the page.
func TestInstanceValidateRejectsDuplicateFlag(t *testing.T) {
	t.Parallel()

	c := validInstance()
	c.Flags = []cloudsql.Flag{
		{Name: flagLogicalDecoding, Value: "on"},
		{Name: flagLogicalDecoding, Value: "off"},
	}

	if err := c.Validate(); err == nil {
		t.Fatal("Validate() = nil, want a duplicate-flag error")
	}
}

func TestDatabaseValidate(t *testing.T) {
	t.Parallel()

	if err := (cloudsql.DatabaseConfig{Name: dbApp}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := (cloudsql.DatabaseConfig{}).Validate(); !errors.Is(err, cloudsql.ErrDatabaseNameRequired) {
		t.Errorf("Validate() = %v, want ErrDatabaseNameRequired", err)
	}
}

func TestValidateDatabasesRejectsDuplicate(t *testing.T) {
	t.Parallel()

	dbs := []cloudsql.DatabaseConfig{{Name: dbApp}, {Name: dbApp}}
	if err := cloudsql.ValidateDatabases(dbs); !errors.Is(err, cloudsql.ErrDuplicateDatabaseName) {
		t.Errorf("ValidateDatabases() = %v, want ErrDuplicateDatabaseName", err)
	}
}

// An empty slice is legitimate: a stack that creates an instance and no
// database yet must not be an error.
func TestValidateDatabasesAcceptsEmpty(t *testing.T) {
	t.Parallel()

	if err := cloudsql.ValidateDatabases(nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user cloudsql.UserConfig
		want error
	}{
		{"built-in with password", cloudsql.UserConfig{Name: userSvc, Password: "p"}, nil},
		{
			"explicit built-in with password",
			cloudsql.UserConfig{Name: userSvc, Type: cloudsql.UserTypeBuiltIn, Password: "p"},
			nil,
		},
		{
			"iam user",
			cloudsql.UserConfig{Name: userPerson, Type: cloudsql.UserTypeIAMUser},
			nil,
		},
		{
			"iam service account",
			cloudsql.UserConfig{Name: "robot@example-project.iam", Type: cloudsql.UserTypeIAMServiceAccount},
			nil,
		},
		{"missing name", cloudsql.UserConfig{Password: "p"}, cloudsql.ErrUserNameRequired},
		{"built-in without password", cloudsql.UserConfig{Name: userSvc}, cloudsql.ErrUserPasswordRequired},
		{
			"iam user carrying a password",
			cloudsql.UserConfig{Name: userPerson, Type: cloudsql.UserTypeIAMUser, Password: "p"},
			cloudsql.ErrUserPasswordNotAllowed,
		},
		{"unknown type", cloudsql.UserConfig{Name: userSvc, Type: "CLOUD_IAM_GROUP_MEMBER"}, cloudsql.ErrUserTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.user.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}

				return
			}

			if !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidateUsersRejectsDuplicate(t *testing.T) {
	t.Parallel()

	users := []cloudsql.UserConfig{
		{Name: userSvc, Password: "a"},
		{Name: userSvc, Password: "b"},
	}
	if err := cloudsql.ValidateUsers(users); !errors.Is(err, cloudsql.ErrDuplicateUserName) {
		t.Errorf("ValidateUsers() = %v, want ErrDuplicateUserName", err)
	}
}

func TestValidateUsersAcceptsEmpty(t *testing.T) {
	t.Parallel()

	if err := cloudsql.ValidateUsers(nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
