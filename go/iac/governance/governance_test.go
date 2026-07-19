// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package governance_test

import (
	"encoding/json"
	"strings"
	"testing"

	governance "github.com/duizendstra/alexandria/go/iac/governance"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Pulumi config keys and shared fixture values.
const (
	keyParent     = "parent"
	keyRootFolder = "rootFolder"
	keyTier       = "tier"
	keyEnvs       = "environments"

	orgParent    = "organizations/123456"
	rootPlatform = "platform"
	tierStarter  = "starter"
	envDev       = "dev"
	envProd      = "prod"
)

type applyMocks struct{}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (applyMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Copy()
	outputs["folderId"] = resource.NewStringProperty(args.Name + "-id")

	return args.Name + "-id", outputs, nil
}

func (applyMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// setConfig marshals the given key/value pairs into the PULUMI_CONFIG
// environment variable the Pulumi test runtime reads config from. Values
// that are not plain strings are JSON-encoded (RequireObject contract).
func setConfig(t *testing.T, kv map[string]any) {
	t.Helper()

	flat := make(map[string]string, len(kv))
	for k, v := range kv {
		if s, ok := v.(string); ok {
			flat["project:"+k] = s

			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		flat["project:"+k] = string(b)
	}

	blob, err := json.Marshal(flat)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PULUMI_CONFIG", string(blob))
}

func run(t *testing.T) error {
	t.Helper()

	return pulumi.RunErr(governance.Apply, pulumi.WithMocks("project", "stack", applyMocks{}))
}

func TestApply_StarterTierAtFolderScope(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     "folders/456",
		keyRootFolder: rootPlatform,
		keyTier:       tierStarter,
	})

	if err := run(t); err != nil {
		t.Fatalf("Apply (starter at folder scope): %v", err)
	}
}

func TestApply_StarterTierAtOrgScope(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     orgParent,
		keyRootFolder: rootPlatform,
		keyTier:       tierStarter,
	})

	if err := run(t); err != nil {
		t.Fatalf("Apply (starter at org scope): %v", err)
	}
}

func TestApply_StandardTierAtOrgScope(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     orgParent,
		keyRootFolder: rootPlatform,
		keyTier:       "standard",
		keyEnvs:       []string{envDev, envProd},
	})

	if err := run(t); err != nil {
		t.Fatalf("Apply (standard at org scope): %v", err)
	}
}

func TestApply_StandardTierIsDefault(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     "folders/789",
		keyRootFolder: rootPlatform,
		keyEnvs:       []string{envDev, envProd},
	})

	if err := run(t); err != nil {
		t.Fatalf("Apply (default standard): %v", err)
	}
}

func TestApply_EnterpriseTier(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     orgParent,
		keyRootFolder: rootPlatform,
		keyTier:       "enterprise",
		keyEnvs:       []string{envDev, "test", envProd},
		"tagKeys": []map[string]string{
			{"shortName": "environment", "description": "Deployment environment"},
		},
		"billingAccount": "01ABCD-234567-89EFGH",
	})

	if err := run(t); err != nil {
		t.Fatalf("Apply (enterprise): %v", err)
	}
}

func TestApply_InvalidParentFails(t *testing.T) {
	setConfig(t, map[string]any{
		keyParent:     "projects/999",
		keyRootFolder: rootPlatform,
		keyTier:       tierStarter,
	})

	err := run(t)
	if err == nil {
		t.Fatal("Apply with invalid parent: want error, got nil")
	}
	if !strings.Contains(err.Error(), "governance:") {
		t.Fatalf("error %v not wrapped with governance: prefix", err)
	}
}
