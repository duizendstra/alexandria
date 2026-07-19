package firestore_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/firestore"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type mocks int

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	return args.Name + "_id", args.Inputs, nil
}

func (mocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

func TestApplyDatabaseCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out, err := firestore.ApplyDatabase(ctx, pulumi.String("proj").ToStringOutput(), firestore.DatabaseConfig{
			Name:             "appdb",
			Region:           "europe-west4",
			DeleteProtection: "DELETE_PROTECTION_ENABLED",
		}, nil)
		if err != nil {
			return err
		}
		if out == nil {
			t.Error("expected outputs")
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyDatabaseInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := firestore.ApplyDatabase(ctx, pulumi.String("proj").ToStringOutput(), firestore.DatabaseConfig{}, nil)
		if !errors.Is(err, firestore.ErrNameRequired) {
			t.Errorf("expected ErrNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyDocumentsSeeds(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return firestore.ApplyDocuments(ctx, pulumi.String("proj").ToStringOutput(), "appdb", []firestore.DocumentConfig{
			{Collection: collectionConfig, DocumentID: docConnectorA, Fields: `{"enabled":{"booleanValue":true}}`},
			{Collection: collectionConfig, DocumentID: "connector-b", Fields: `{"enabled":{"booleanValue":false}}`},
		}, nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyDocumentsInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := firestore.ApplyDocuments(ctx, pulumi.String("proj").ToStringOutput(), "appdb", []firestore.DocumentConfig{
			{DocumentID: docConnectorA, Fields: "f"},
		}, nil)
		if !errors.Is(err, firestore.ErrCollectionRequired) {
			t.Errorf("expected ErrCollectionRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
