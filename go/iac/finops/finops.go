package finops

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/budgets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defaultRegion is used when no "region" config is set.
const defaultRegion = "europe-west4"

// defaultCurrency is used when no "currency" config is set.
const defaultCurrency = "EUR"

// Params allows upstream BCs to pass values directly (collapsed mode).
// When nil, all values come from Pulumi stack config (enterprise mode).
type Params struct {
	// FolderID overrides config — used when governance passes it directly.
	FolderID string
	// BillingAccount overrides config.
	BillingAccount string
	// OrgID overrides config — scope for org-level budgets.
	OrgID string
}

// placement is where the finops project lands, who pays, and the budget scope.
type placement struct {
	folderID       string
	billingAccount string
	orgID          string
}

// Apply creates finops resources: project, billing dataset, budget.
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
			"monitoring.googleapis.com",
			"billingbudgets.googleapis.com",
		},
	})
	if err != nil {
		return fmt.Errorf("finops project: %w", err)
	}

	datasetOutputs, err := datasets.Apply(ctx, projectOutputs.ProjectID, datasets.Config{
		ID:           "billing_export",
		FriendlyName: "Billing Export",
		Description:  "GCP billing data — enable export in Cloud Console",
		Location:     region,
	}, nil)
	if err != nil {
		return fmt.Errorf("billing dataset: %w", err)
	}

	if err := applyBudget(ctx, cfg, projectName, place, projectOutputs.ProjectID); err != nil {
		return err
	}

	ctx.Export("projectId", projectOutputs.ProjectID)
	ctx.Export("billingDatasetId", datasetOutputs.DatasetID)

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
// stack named by the "governanceStack" config key. Best-effort: a missing
// key, unreadable stack, or absent output leaves the value empty so the
// config fallback applies.
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
		if out, gErr := gov.GetOutputDetails(exports.RootFolderID); gErr == nil && out.Value != nil {
			place.folderID, _ = out.Value.(string)
		}
	}
}

// applyBudget creates the org-level budget from stack config: required
// monthly amount and alert emails, optional currency and thresholds.
func applyBudget(ctx *pulumi.Context, cfg *config.Config, projectName string, place placement, projectID pulumi.StringOutput) error {
	monthlyBudget := cfg.RequireInt("monthlyBudget")

	currency := cfg.Get("currency")
	if currency == "" {
		currency = defaultCurrency
	}

	var alertEmails []string
	cfg.RequireObject("alertEmails", &alertEmails)

	thresholds := []float64{0.50, 0.75, 0.90, 1.00}

	var customThresholds []float64
	if err := cfg.TryObject("thresholds", &customThresholds); err == nil && len(customThresholds) > 0 {
		thresholds = customThresholds
	}

	budgetCfg := &budgets.Config{
		DisplayName:    fmt.Sprintf("%s: %d %s/month", projectName, monthlyBudget, currency),
		Amount:         monthlyBudget,
		Currency:       currency,
		BillingAccount: place.billingAccount,
		Scope:          "organizations/" + place.orgID,
		Thresholds:     thresholds,
		AlertEmails:    alertEmails,
	}

	if err := budgets.Apply(ctx, projectID, budgetCfg, nil); err != nil {
		return fmt.Errorf("budget: %w", err)
	}

	return nil
}

// FinOps runs finops as a standalone Pulumi program.
// Enterprise mode — reads everything from config.
func FinOps() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
