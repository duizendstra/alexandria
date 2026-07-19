package observability

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/logsinks"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defaultRegion is used when no "region" config is set.
const defaultRegion = "europe-west4"

// defaultGovernanceFolder is the folder key read from the governance
// stack's folder-ID export map when no "governanceFolder" config is set.
const defaultGovernanceFolder = "shared"

// Params allows upstream BCs to pass values directly (collapsed mode).
// When nil, all values come from Pulumi stack config (enterprise mode).
type Params struct {
	// FolderID overrides config — used when governance passes it directly.
	FolderID string
	// BillingAccount overrides config.
	BillingAccount string
	// OrgID overrides config — source scope for the org log sink.
	OrgID string
}

// placement is where the project lands, who pays, and the sink's org scope.
type placement struct {
	folderID       string
	billingAccount string
	orgID          string
}

// Apply creates observability resources: project, log dataset, org sink.
//
// Resolution order for folderID, billingAccount, orgID:
//  1. Params (collapsed mode — governance passes directly)
//  2. Governance stack reference named by the "governanceStack" config key
//  3. Stack config (explicit override / fallback)
func Apply(ctx *pulumi.Context, params *Params) error {
	cfg := config.New(ctx, "")

	projectName := cfg.Require("projectName")

	region := cfg.Get("region")
	if region == "" {
		region = defaultRegion
	}

	place := resolvePlacement(ctx, cfg, params)

	projectOutputs, err := projects.Apply(ctx, projects.Config{
		Name:           projectName,
		FolderID:       place.folderID,
		BillingAccount: place.billingAccount,
		APIs: []string{
			"bigquery.googleapis.com",
			"logging.googleapis.com",
			"monitoring.googleapis.com",
		},
	})
	if err != nil {
		return fmt.Errorf("observability project: %w", err)
	}

	datasetOutputs, err := datasets.Apply(ctx, projectOutputs.ProjectID, datasets.Config{
		ID:           "org_logs",
		FriendlyName: "Org-wide Audit Logs",
		Description:  "Aggregated audit and activity logs from all GCP projects",
		Location:     region,
	}, nil)
	if err != nil {
		return fmt.Errorf("log dataset: %w", err)
	}

	destination := pulumi.Sprintf(
		"bigquery.googleapis.com/projects/%s/datasets/%s",
		projectOutputs.ProjectID, datasetOutputs.DatasetID,
	)

	sinkOutputs, err := logsinks.Apply(ctx, logsinks.Config{
		Name:            "org-audit-to-bigquery",
		OrgID:           place.orgID,
		Filter:          `logName:"logs/cloudaudit.googleapis.com"`,
		IncludeChildren: true,
	}, destination)
	if err != nil {
		return fmt.Errorf("org log sink: %w", err)
	}

	// Sink writer identity exported so downstream can grant BQ access.
	ctx.Export("projectId", projectOutputs.ProjectID)
	ctx.Export("logDatasetId", datasetOutputs.DatasetID)
	ctx.Export("sinkWriterIdentity", sinkOutputs.WriterIdentity)

	return nil
}

// resolvePlacement determines folder ID, billing account, and org ID from
// params, the governance stack reference, and stack config, in that order.
func resolvePlacement(ctx *pulumi.Context, cfg *config.Config, params *Params) placement {
	place := placement{}
	if params != nil {
		place.folderID = params.FolderID
		place.billingAccount = params.BillingAccount
		place.orgID = params.OrgID
	}

	if place.folderID == "" || place.billingAccount == "" || place.orgID == "" {
		fromGovernanceStack(ctx, cfg, &place)
	}

	if place.folderID == "" {
		place.folderID = cfg.Require("folderID")
	}

	if place.billingAccount == "" {
		place.billingAccount = cfg.Require("billingAccount")
	}

	if place.orgID == "" {
		place.orgID = cfg.Require("orgID")
	}

	return place
}

// fromGovernanceStack fills missing placement values from the governance
// stack named by the "governanceStack" config key; the folder is chosen
// by the optional "governanceFolder" key (default "shared").
// Best-effort: a missing key, unreadable stack, or absent output leaves
// the value empty so the config fallback applies.
func fromGovernanceStack(ctx *pulumi.Context, cfg *config.Config, place *placement) {
	govRef := cfg.Get("governanceStack")
	if govRef == "" {
		return
	}

	gov, err := pulumi.NewStackReference(ctx, govRef, nil)
	if err != nil {
		return
	}

	if place.billingAccount == "" {
		if out, gErr := gov.GetOutputDetails(exports.BillingAccount); gErr == nil && out.Value != nil {
			place.billingAccount, _ = out.Value.(string)
		}
	}

	if place.orgID == "" {
		if out, gErr := gov.GetOutputDetails(exports.OrgID); gErr == nil && out.Value != nil {
			place.orgID, _ = out.Value.(string)
		}
	}

	if place.folderID == "" {
		folder := cfg.Get("governanceFolder")
		if folder == "" {
			folder = defaultGovernanceFolder
		}

		if out, gErr := gov.GetOutputDetails(exports.FolderIDs); gErr == nil && out.Value != nil {
			if m, ok := out.Value.(map[string]any); ok {
				place.folderID, _ = m[folder].(string)
			}
		}
	}
}

// Observability runs observability as a standalone Pulumi program.
// Enterprise mode — reads everything from config.
func Observability() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
