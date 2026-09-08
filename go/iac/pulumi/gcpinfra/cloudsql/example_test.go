// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package cloudsql_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudsql"
)

func ExampleInstanceConfig_Validate() {
	c := cloudsql.InstanceConfig{
		Name:            "shared-postgres",
		Region:          "europe-west4",
		DatabaseVersion: "POSTGRES_18",
		Tier:            "db-custom-1-3840",
		DiskSizeGB:      10,
		Flags: []cloudsql.Flag{
			{Name: flagLogicalDecoding, Value: "on"},
			{Name: "cloudsql.iam_authentication", Value: "on"},
		},
		SSLMode:       "ENCRYPTED_ONLY",
		BackupEnabled: true,
	}
	fmt.Println(c.Validate())
	// Output:
	// <nil>
}

// An IAM user authenticates as a Google Cloud identity and has no password of
// its own; supplying one is a configuration mistake rather than a stronger
// setting, so it is refused.
func ExampleUserConfig_Validate() {
	iam := cloudsql.UserConfig{Name: userPerson, Type: cloudsql.UserTypeIAMUser}
	fmt.Println(iam.Validate())

	withPassword := cloudsql.UserConfig{
		Name:     userPerson,
		Type:     cloudsql.UserTypeIAMUser,
		Password: "hunter2",
	}
	fmt.Println(withPassword.Validate())
	// Output:
	// <nil>
	// cloudsql: IAM user must not carry a Password: "person@example.invalid"
}
