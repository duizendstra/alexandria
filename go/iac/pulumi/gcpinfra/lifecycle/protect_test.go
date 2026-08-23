// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package lifecycle_test

import (
	"sort"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/firestore"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/folders"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/registries"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tables"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tagkeys"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// The protection policy is module-wide, so it is asserted in one place rather
// than a piece at a time in each building block: a block added later without
// a decision about its data shows up here as an unexpected resource, which a
// per-package test could never notice.
//
// The map is keyed by type token and logical name together. Native and
// external BigQuery tables share one type token and hold opposite policies —
// the rows live in the table in one case and in the source in the other.
var (
	wantProtected = map[string]bool{
		"gcp:organizations/folder:Folder::root":                            true,
		"gcp:organizations/folder:Folder::dev":                             true,
		"gcp:tags/tagKey:TagKey::env":                                      true,
		"gcp:organizations/project:Project::example-project":               true,
		"gcp:projects/service:Service::example-project-run.googleapis.com": false,
		"gcp:bigquery/dataset:Dataset::warehouse":                          true,
		"gcp:bigquery/table:Table::events":                                 true,
		"gcp:bigquery/table:Table::sheet":                                  false,
		"gcp:firestore/database:Database::firestore-appdb":                 true,
		"gcp:firestore/document:Document::doc-connector":                   true,
		"gcp:artifactregistry/repository:Repository::images":               true,
		"gcp:secretmanager/secret:Secret::api-key":                         true,
		"gcp:secretmanager/secretVersion:SecretVersion::api-key-v1":        false,
	}
)

// region is the one location the whole fixture stack is built in.
const region = "europe-west4"

type record struct {
	key     string
	protect bool
}

type recorder struct {
	mu      sync.Mutex
	records []record
}

func (r *recorder) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	r.mu.Lock()
	r.records = append(r.records, record{
		key:     args.TypeToken + "::" + args.Name,
		protect: args.RegisterRPC.GetProtect(),
	})
	r.mu.Unlock()

	outputs := args.Inputs.Copy()
	outputs["name"] = resource.NewStringProperty(args.Name)

	return args.Name + "-id", outputs, nil
}

func (r *recorder) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// applyAll runs every building block that creates a resource this policy
// covers, so one stack answers for the whole module.
func applyAll(ctx *pulumi.Context, opts ...lifecycle.Option) error {
	projectID := pulumi.String("example-project").ToStringOutput()

	if _, err := folders.Apply(ctx, hierarchy.Config{
		Parent:   "organizations/1",
		RootName: "root",
		Children: []string{"dev"},
	}, opts...); err != nil {
		return err
	}

	if _, err := tagkeys.Apply(ctx, "1", []classification.Dimension{
		{ShortName: "env", Description: "environment"},
	}, opts...); err != nil {
		return err
	}

	if _, err := projects.Apply(ctx, projects.Config{
		Name:           "example-project",
		FolderID:       "1",
		BillingAccount: "012345-6789AB-CDEF01",
		APIs:           []string{"run.googleapis.com"},
	}, opts...); err != nil {
		return err
	}

	ds, err := datasets.Apply(ctx, projectID, datasets.Config{
		ID:           "warehouse",
		FriendlyName: "Warehouse",
		Location:     region,
	}, nil, opts...)
	if err != nil {
		return err
	}

	if err := tables.Apply(ctx, projectID, ds.DatasetID, []tables.Config{
		{Name: "events", Schema: "[]"},
	}, nil, opts...); err != nil {
		return err
	}

	if err := tables.ApplyExternal(ctx, projectID, ds.DatasetID, []tables.ExternalConfig{
		{Name: "sheet", SourceFormat: "GOOGLE_SHEETS", SourceURIs: []string{"https://example.invalid/s"}},
	}, nil, opts...); err != nil {
		return err
	}

	if _, err := firestore.ApplyDatabase(ctx, projectID, firestore.DatabaseConfig{
		Name:   "appdb",
		Region: region,
	}, nil, opts...); err != nil {
		return err
	}

	if err := firestore.ApplyDocuments(ctx, projectID, "appdb", []firestore.DocumentConfig{
		{Collection: "config", DocumentID: "connector", Fields: "{}"},
	}, nil, opts...); err != nil {
		return err
	}

	if _, err := registries.Apply(ctx, projectID, registries.Config{
		ID:       "images",
		Format:   "DOCKER",
		Location: region,
	}, nil, opts...); err != nil {
		return err
	}

	return secrets.Apply(ctx, projectID, []secrets.Secret{
		{Name: "api-key", Value: "v"},
	}, nil, opts...)
}

func runStack(t *testing.T, opts ...lifecycle.Option) map[string]bool {
	t.Helper()

	rec := &recorder{}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return applyAll(ctx, opts...)
	}, pulumi.WithMocks("project", "stack", rec))
	if err != nil {
		t.Fatalf("RunErr: %v", err)
	}

	got := make(map[string]bool, len(rec.records))
	for _, r := range rec.records {
		got[r.key] = r.protect
	}

	return got
}

// assertPolicy compares the whole stack against want, so a resource that goes
// missing fails as loudly as one whose protection flipped.
func assertPolicy(t *testing.T, got, want map[string]bool) {
	t.Helper()

	keys := make([]string, 0, len(want)+len(got))
	for k := range want {
		keys = append(keys, k)
	}

	for k := range got {
		if _, ok := want[k]; !ok {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	for _, k := range keys {
		w, wanted := want[k]
		g, created := got[k]

		switch {
		case !created:
			t.Errorf("%s: never created — the policy for it is asserting nothing", k)
		case !wanted:
			t.Errorf("%s: created but carries no policy — decide whether it holds data", k)
		case g != w:
			t.Errorf("%s: protect = %v, want %v", k, g, w)
		}
	}
}

func TestDataBearingResourcesAreProtectedByDefault(t *testing.T) {
	t.Parallel()

	assertPolicy(t, runStack(t), wantProtected)
}

func TestEphemeralClearsProtectionOnEveryResource(t *testing.T) {
	t.Parallel()

	want := make(map[string]bool, len(wantProtected))
	for k := range wantProtected {
		want[k] = false
	}

	assertPolicy(t, runStack(t, lifecycle.Ephemeral()), want)
}

func TestIsEphemeral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []lifecycle.Option
		want bool
	}{
		{name: "no options", opts: nil, want: false},
		{name: "ephemeral", opts: []lifecycle.Option{lifecycle.Ephemeral()}, want: true},
		{name: "nil option is ignored", opts: []lifecycle.Option{nil}, want: false},
		{name: "repeated", opts: []lifecycle.Option{lifecycle.Ephemeral(), lifecycle.Ephemeral()}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := lifecycle.IsEphemeral(tt.opts...); got != tt.want {
				t.Errorf("IsEphemeral() = %v, want %v", got, tt.want)
			}
		})
	}
}
