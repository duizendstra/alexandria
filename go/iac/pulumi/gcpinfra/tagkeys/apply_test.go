// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package tagkeys_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tagkeys"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const dimEnv = "env"

type applyMocks struct{}

func (applyMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	return args.Name + "-id", args.Inputs.Copy(), nil
}

func (applyMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestApply_RequiresOrgID(t *testing.T) {
	t.Parallel()

	// Validation fails before any resource is created; nil ctx is safe.
	_, err := tagkeys.Apply(nil, "", []classification.Dimension{{ShortName: dimEnv, Description: "d"}})
	if !errors.Is(err, tagkeys.ErrOrgIDRequired) {
		t.Fatalf("Apply error = %v, want ErrOrgIDRequired", err)
	}
}

func TestApply_RejectsInvalidDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dims    []classification.Dimension
		wantErr error
	}{
		{name: "empty set", dims: nil, wantErr: classification.ErrNoDimensions},
		{
			name:    "missing short name",
			dims:    []classification.Dimension{{Description: "d"}},
			wantErr: classification.ErrShortNameRequired,
		},
		{
			name: "duplicate short name",
			dims: []classification.Dimension{
				{ShortName: dimEnv, Description: "a"},
				{ShortName: dimEnv, Description: "b"},
			},
			wantErr: classification.ErrDuplicateShortName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tagkeys.Apply(nil, "123", tt.dims)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Apply error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestApply_CreatesTagKeys(t *testing.T) {
	t.Parallel()

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := tagkeys.Apply(ctx, "123456", []classification.Dimension{
			{ShortName: "environment", Description: "Deployment environment"},
			{ShortName: "data-class", Description: "Data sensitivity"},
		})
		if err != nil {
			return err
		}
		if len(out) != 2 {
			t.Fatalf("outputs has %d entries, want 2", len(out))
		}
		for _, name := range []string{"environment", "data-class"} {
			if _, ok := out[name]; !ok {
				t.Fatalf("missing output for dimension %q (got %v)", name, out)
			}
		}

		return nil
	}, pulumi.WithMocks("project", "stack", applyMocks{}))
	if err != nil {
		t.Fatalf("RunErr: %v", err)
	}
}
