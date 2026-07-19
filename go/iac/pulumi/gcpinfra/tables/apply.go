package tables

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/bigquery"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Apply creates BigQuery tables with schema in the given dataset.
func Apply(ctx *pulumi.Context, projectID, datasetID pulumi.StringOutput, tt []Config, deps []pulumi.Resource) error {
	for i := range tt {
		t := &tt[i]
		if err := t.Validate(); err != nil {
			return err
		}

		labels := make(pulumi.StringMap)
		for k, v := range t.Labels {
			labels[k] = pulumi.String(v)
		}

		args := &bigquery.TableArgs{
			Project:            projectID,
			DatasetId:          datasetID,
			TableId:            pulumi.String(t.Name),
			Schema:             pulumi.String(t.Schema),
			DeletionProtection: pulumi.Bool(t.DeletionProtection),
			Labels:             labels,
		}

		if t.PartitionField != "" {
			args.TimePartitioning = &bigquery.TableTimePartitioningArgs{
				Type:  pulumi.String("DAY"),
				Field: pulumi.String(t.PartitionField),
			}
		}

		_, err := bigquery.NewTable(ctx, t.Name, args, pulumi.DependsOn(deps))
		if err != nil {
			return fmt.Errorf("create table %s: %w", t.Name, err)
		}
	}

	return nil
}

// ApplyExternal creates BigQuery external tables (e.g. Google Sheets).
func ApplyExternal(ctx *pulumi.Context, projectID, datasetID pulumi.StringOutput, tt []ExternalConfig, deps []pulumi.Resource) error {
	for i := range tt {
		t := &tt[i]
		if err := t.Validate(); err != nil {
			return err
		}

		labels := make(pulumi.StringMap)
		for k, v := range t.Labels {
			labels[k] = pulumi.String(v)
		}

		uris := make(pulumi.StringArray, len(t.SourceURIs))
		for j, u := range t.SourceURIs {
			uris[j] = pulumi.String(u)
		}

		extCfg := &bigquery.TableExternalDataConfigurationArgs{
			Autodetect:   pulumi.Bool(false),
			SourceFormat: pulumi.String(t.SourceFormat),
			SourceUris:   uris,
			Compression:  pulumi.String("NONE"),
		}

		if t.SourceFormat == "GOOGLE_SHEETS" {
			extCfg.GoogleSheetsOptions = &bigquery.TableExternalDataConfigurationGoogleSheetsOptionsArgs{
				Range:           pulumi.String(t.SheetRange),
				SkipLeadingRows: pulumi.Int(t.SkipLeadingRows),
			}
		}

		args := &bigquery.TableArgs{
			Project:                   projectID,
			DatasetId:                 datasetID,
			TableId:                   pulumi.String(t.Name),
			DeletionProtection:        pulumi.Bool(t.DeletionProtection),
			Labels:                    labels,
			ExternalDataConfiguration: extCfg,
		}

		if t.Schema != "" {
			args.Schema = pulumi.String(t.Schema)
		}

		_, err := bigquery.NewTable(ctx, t.Name, args, pulumi.DependsOn(deps))
		if err != nil {
			return fmt.Errorf("create external table %s: %w", t.Name, err)
		}
	}

	return nil
}
