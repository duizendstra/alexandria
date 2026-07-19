// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package folders_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/governance/hierarchy"
	"github.com/duizendstra/alexandria/go/governance/scope"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/folders"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	orgParent    = "organizations/123456"
	folderParent = "folders/789012"
	rootName     = "root"
	childDev     = "dev"
)

func TestParseScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parent  string
		want    scope.Scope
		wantErr bool
	}{
		{name: "organization", parent: orgParent, want: scope.Organization},
		{name: "folder", parent: folderParent, want: scope.Container},
		{name: "organization no id", parent: "organizations/", want: scope.Organization},
		{name: "empty", parent: "", wantErr: true},
		{name: "project", parent: "projects/123", wantErr: true},
		{name: "bare id", parent: "123456", wantErr: true},
		{name: "prefix without slash", parent: "organizations", wantErr: true},
		{name: "case sensitive", parent: "Organizations/123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := folders.ParseScope(tt.parent)
			if tt.wantErr {
				if !errors.Is(err, folders.ErrInvalidParent) {
					t.Fatalf("ParseScope(%q) error = %v, want ErrInvalidParent", tt.parent, err)
				}

				return
			}
			if err != nil {
				t.Fatalf("ParseScope(%q): %v", tt.parent, err)
			}
			if got != tt.want {
				t.Fatalf("ParseScope(%q) = %v, want %v", tt.parent, got, tt.want)
			}
		})
	}
}

func FuzzParseScope(f *testing.F) {
	f.Add(orgParent)
	f.Add(folderParent)
	f.Add("projects/1")
	f.Add("")
	f.Add("organizations/")

	f.Fuzz(func(t *testing.T, parent string) {
		got, err := folders.ParseScope(parent)

		// Invariant: success iff the parent carries a known prefix, and the
		// scope must match that prefix.
		switch {
		case strings.HasPrefix(parent, "organizations/"):
			if err != nil || got != scope.Organization {
				t.Fatalf("ParseScope(%q) = (%v, %v), want (Organization, nil)", parent, got, err)
			}
		case strings.HasPrefix(parent, "folders/"):
			if err != nil || got != scope.Container {
				t.Fatalf("ParseScope(%q) = (%v, %v), want (Container, nil)", parent, got, err)
			}
		default:
			if !errors.Is(err, folders.ErrInvalidParent) {
				t.Fatalf("ParseScope(%q) error = %v, want ErrInvalidParent", parent, err)
			}
		}
	})
}

func TestOrgID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		parent string
		want   string
	}{
		{parent: orgParent, want: "123456"},
		// Non-org parents pass through unchanged; callers gate on scope first.
		{parent: folderParent, want: folderParent},
		{parent: "", want: ""},
	}

	for _, tt := range tests {
		if got := folders.OrgID(tt.parent); got != tt.want {
			t.Fatalf("OrgID(%q) = %q, want %q", tt.parent, got, tt.want)
		}
	}
}

// applyMocks satisfies pulumi.MockResourceMonitor for offline Apply tests.
type applyMocks struct{}

func (applyMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	outputs := args.Inputs.Copy()
	outputs["name"] = resource.NewStringProperty(args.Name)
	outputs["folderId"] = resource.NewStringProperty(args.Name + "-id")

	return args.Name + "-id", outputs, nil
}

func (applyMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestApply_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     hierarchy.Config
		wantErr error
	}{
		{
			name:    "missing parent",
			cfg:     hierarchy.Config{RootName: rootName},
			wantErr: hierarchy.ErrParentRequired,
		},
		{
			name:    "missing root name",
			cfg:     hierarchy.Config{Parent: "organizations/1"},
			wantErr: hierarchy.ErrRootNameRequired,
		},
		{
			name:    "duplicate children",
			cfg:     hierarchy.Config{Parent: "organizations/1", RootName: rootName, Children: []string{childDev, childDev}},
			wantErr: hierarchy.ErrDuplicateChild,
		},
		{
			name:    "invalid parent format",
			cfg:     hierarchy.Config{Parent: "projects/1", RootName: rootName},
			wantErr: folders.ErrInvalidParent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Validation fails before any resource is created, so a nil
			// Pulumi context is never dereferenced.
			_, err := folders.Apply(nil, tt.cfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Apply error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestApply_CreatesHierarchy(t *testing.T) {
	t.Parallel()

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := folders.Apply(ctx, hierarchy.Config{
			Parent:   "organizations/123",
			RootName: "platform",
			Children: []string{childDev, "prod"},
		})
		if err != nil {
			return err
		}
		if out == nil {
			t.Fatal("Apply returned nil outputs")
		}
		if len(out.FolderIDs) != 2 {
			t.Fatalf("FolderIDs has %d entries, want 2", len(out.FolderIDs))
		}

		return nil
	}, pulumi.WithMocks("project", "stack", applyMocks{}))
	if err != nil {
		t.Fatalf("RunErr: %v", err)
	}
}

// TestApply_StarterShapeWithoutChildren pins the fix for the tier-policy
// leak: a Starter hierarchy has no children by definition
// (plan.validateStarter rejects them), so the adapter must not require any.
// Before the fix, Apply called cfg.Validate(), which returns ErrNoChildren —
// making every Starter deployment fail.
func TestApply_StarterShapeWithoutChildren(t *testing.T) {
	t.Parallel()

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := folders.Apply(ctx, hierarchy.Config{
			Parent:   "organizations/123",
			RootName: "platform",
		})
		if err != nil {
			return err
		}
		if len(out.FolderIDs) != 0 {
			t.Fatalf("FolderIDs has %d entries, want 0 for starter shape", len(out.FolderIDs))
		}

		return nil
	}, pulumi.WithMocks("project", "stack", applyMocks{}))
	if err != nil {
		t.Fatalf("RunErr: %v", err)
	}
}
