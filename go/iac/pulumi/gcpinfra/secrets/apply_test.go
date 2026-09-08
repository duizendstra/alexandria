// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package secrets_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	tokenSecret  = "gcp:secretmanager/secret:Secret"
	tokenVersion = "gcp:secretmanager/secretVersion:SecretVersion"

	secretName = "api-key"

	containerName = "hand-written-password"
)

// registry keeps every resource the monitor was asked to create, so a test can
// assert on what was NOT created as well as on what was.
type registry struct {
	mu    sync.Mutex
	types []string
	names map[string]resource.PropertyMap
}

func (r *registry) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	r.mu.Lock()
	if r.names == nil {
		r.names = map[string]resource.PropertyMap{}
	}

	r.types = append(r.types, args.TypeToken)
	r.names[args.TypeToken+"::"+args.Name] = args.Inputs.Copy()
	r.mu.Unlock()

	return args.Name + "-id", args.Inputs, nil
}

func (r *registry) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (r *registry) created(token string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := 0

	for _, t := range r.types {
		if t == token {
			n++
		}
	}

	return n
}

func run(t *testing.T, body func(ctx *pulumi.Context) error) *registry {
	t.Helper()

	reg := &registry{}
	if err := pulumi.RunErr(body, pulumi.WithMocks("example-project", "stack", reg)); err != nil {
		t.Fatalf("pulumi run: %v", err)
	}

	return reg
}

// TestApplyContainersCreatesNoVersion is the whole point of the container
// shape: the stack creates the secret and never learns its value. A version
// here would mean the value passed through Pulumi and is in the state file.
func TestApplyContainersCreatesNoVersion(t *testing.T) {
	reg := run(t, func(ctx *pulumi.Context) error {
		return secrets.ApplyContainers(ctx, pulumi.String("example-project").ToStringOutput(),
			[]secrets.Container{{Name: containerName, Labels: map[string]string{"env": "test"}}}, nil)
	})

	if got := reg.created(tokenSecret); got != 1 {
		t.Errorf("created %d secrets, want 1", got)
	}

	if got := reg.created(tokenVersion); got != 0 {
		t.Errorf("created %d secret versions, want none: the value must never pass through the stack", got)
	}
}

// TestApplyCreatesAVersion is the contrast that keeps the test above honest:
// the value-carrying Apply does create one, so a zero there would be a broken
// monitor rather than a kept promise.
func TestApplyCreatesAVersion(t *testing.T) {
	reg := run(t, func(ctx *pulumi.Context) error {
		return secrets.Apply(ctx, pulumi.String("example-project").ToStringOutput(),
			[]secrets.Secret{{Name: secretName, Value: "v"}}, nil)
	})

	if got := reg.created(tokenVersion); got != 1 {
		t.Errorf("created %d secret versions, want 1", got)
	}
}

func TestApplyContainersDeletionPolicy(t *testing.T) {
	tests := []struct {
		name string
		opts []lifecycle.Option
		want string
	}{
		{name: "durable prevents deletion", want: "PREVENT"},
		{name: "ephemeral allows it", opts: []lifecycle.Option{lifecycle.Ephemeral()}, want: "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := run(t, func(ctx *pulumi.Context) error {
				return secrets.ApplyContainers(ctx, pulumi.String("example-project").ToStringOutput(),
					[]secrets.Container{{Name: containerName}}, nil, tt.opts...)
			})

			inputs := reg.names[tokenSecret+"::"+containerName]

			got, ok := inputs["deletionPolicy"]
			if !ok {
				t.Fatalf("no deletionPolicy sent; sent: %v", inputs.Mappable())
			}

			if got.StringValue() != tt.want {
				t.Errorf("deletionPolicy = %q, want %q", got.StringValue(), tt.want)
			}

			protection, ok := inputs["deletionProtection"]
			if !ok {
				t.Fatalf("no deletionProtection sent; sent: %v", inputs.Mappable())
			}

			if protection.BoolValue() != (tt.want == "PREVENT") {
				t.Errorf("deletionProtection = %v alongside deletionPolicy %q", protection, tt.want)
			}
		})
	}
}

func TestValidateContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		containers []secrets.Container
		wantErr    error
	}{
		{name: "empty list", wantErr: secrets.ErrNoContainers},
		{name: "missing name", containers: []secrets.Container{{}}, wantErr: secrets.ErrNameRequired},
		{
			name:       "duplicate name",
			containers: []secrets.Container{{Name: containerName}, {Name: containerName}},
			wantErr:    secrets.ErrDuplicateName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := secrets.ValidateContainers(tt.containers)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyContainersRejectsDuplicates(t *testing.T) {
	run(t, func(ctx *pulumi.Context) error {
		err := secrets.ApplyContainers(ctx, pulumi.String("example-project").ToStringOutput(),
			[]secrets.Container{{Name: containerName}, {Name: containerName}}, nil)
		if !errors.Is(err, secrets.ErrDuplicateName) {
			t.Errorf("expected ErrDuplicateName, got %v", err)
		}

		if err == nil || !strings.Contains(err.Error(), containerName) {
			t.Errorf("error should name the duplicate, got %v", err)
		}

		return nil
	})
}
