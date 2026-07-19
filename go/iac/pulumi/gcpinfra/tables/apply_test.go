package tables_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tables"
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

const (
	tableEvents   = "events"
	tableForecast = "forecast"
	formatSheets  = "GOOGLE_SHEETS"
)

const schema = `[{"name":"id","type":"STRING","mode":"REQUIRED"},{"name":"load_dts","type":"TIMESTAMP","mode":"NULLABLE"}]`

func TestApplyCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return tables.Apply(ctx, pulumi.String("proj").ToStringOutput(), pulumi.String("landing").ToStringOutput(), []tables.Config{
			{Name: tableEvents, Schema: schema, PartitionField: "load_dts", Labels: map[string]string{"env": "test"}},
			{Name: "lookup", Schema: schema},
		}, nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := tables.Apply(ctx, pulumi.String("proj").ToStringOutput(), pulumi.String("landing").ToStringOutput(), []tables.Config{
			{Name: tableEvents},
		}, nil)
		if !errors.Is(err, tables.ErrSchemaRequired) {
			t.Errorf("expected ErrSchemaRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyExternalCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return tables.ApplyExternal(ctx, pulumi.String("proj").ToStringOutput(), pulumi.String("landing").ToStringOutput(), []tables.ExternalConfig{
			{
				Name:            tableForecast,
				Schema:          schema,
				SourceFormat:    formatSheets,
				SourceURIs:      []string{"https://docs.google.com/spreadsheets/d/example"},
				SheetRange:      "Sheet1!A1:B100",
				SkipLeadingRows: 1,
				Labels:          map[string]string{"env": "test"},
			},
		}, nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyExternalInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := tables.ApplyExternal(ctx, pulumi.String("proj").ToStringOutput(), pulumi.String("landing").ToStringOutput(), []tables.ExternalConfig{
			{Name: tableForecast, SourceFormat: formatSheets},
		}, nil)
		if !errors.Is(err, tables.ErrSourceURIsRequired) {
			t.Errorf("expected ErrSourceURIsRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
