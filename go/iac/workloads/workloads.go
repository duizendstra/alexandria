package workloads

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/iambindings"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// ProjectConfig defines a single project within an environment.
type ProjectConfig struct {
	Name     string   `json:"name"`
	Concerns []string `json:"concerns"` // e.g. compute, data, reports.
	APIs     []string `json:"apis"`
}

// Params allows upstream BCs to pass values directly (collapsed mode).
// When nil, all values come from Pulumi stack config (enterprise mode).
type Params struct {
	// FolderID overrides config — used when governance passes it directly.
	FolderID string
	// BillingAccount overrides config.
	BillingAccount string
}

// placement is where the environment's projects land and who pays.
type placement struct {
	folderID       string
	billingAccount string
}

// Apply creates all projects for a workload environment.
//
// Resolution order for folderID and billingAccount:
//  1. Params (collapsed mode — governance passes directly)
//  2. Governance stack reference named by the "governanceStack" config key
//  3. Stack config (explicit override / fallback)
func Apply(ctx *pulumi.Context, params *Params) error {
	cfg := config.New(ctx, "")

	environmentFolder := cfg.Require("environmentFolder")

	var projectConfigs []ProjectConfig
	cfg.RequireObject("projects", &projectConfigs)

	place := resolvePlacement(ctx, cfg, params, environmentFolder)

	// Delivery trigger SA — granted deploy access to each workload project.
	deliveryProjectNumber := cfg.Get("deliveryProjectNumber")

	for _, pc := range projectConfigs {
		if err := applyProject(ctx, pc, place, deliveryProjectNumber); err != nil {
			return err
		}
	}

	return nil
}

// resolvePlacement determines folder ID and billing account from params,
// the governance stack reference, and stack config, in that order.
func resolvePlacement(ctx *pulumi.Context, cfg *config.Config, params *Params, environmentFolder string) placement {
	place := placement{}
	if params != nil {
		place.folderID = params.FolderID
		place.billingAccount = params.BillingAccount
	}

	if place.folderID == "" || place.billingAccount == "" {
		fromGovernanceStack(ctx, cfg, environmentFolder, &place)
	}

	if place.folderID == "" {
		place.folderID = cfg.Require("folderID")
	}

	if place.billingAccount == "" {
		place.billingAccount = cfg.Require("billingAccount")
	}

	return place
}

// fromGovernanceStack fills missing placement values from the governance
// stack named by the "governanceStack" config key; the folder is chosen
// by environmentFolder from the stack's folder-ID export map.
// Best-effort: a missing key, unreadable stack, or absent output leaves
// the value empty so the config fallback applies.
func fromGovernanceStack(ctx *pulumi.Context, cfg *config.Config, environmentFolder string, place *placement) {
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

	if place.folderID == "" {
		if out, gErr := gov.GetOutputDetails(exports.FolderIDs); gErr == nil && out.Value != nil {
			if m, ok := out.Value.(map[string]any); ok {
				place.folderID, _ = m[environmentFolder].(string)
			}
		}
	}
}

// applyProject creates one project, exports its outputs under each of
// its concerns, and grants the delivery trigger SA deploy access when a
// delivery project number is configured.
func applyProject(ctx *pulumi.Context, pc ProjectConfig, place placement, deliveryProjectNumber string) error {
	outputs, err := projects.Apply(ctx, projects.Config{
		Name:           pc.Name,
		FolderID:       place.folderID,
		BillingAccount: place.billingAccount,
		APIs:           pc.APIs,
	})
	if err != nil {
		return fmt.Errorf("project %s: %w", pc.Name, err)
	}

	// A collapsed project serves multiple concerns — export under each.
	for _, concern := range pc.Concerns {
		ctx.Export(concern+"ProjectId", outputs.ProjectID)
		ctx.Export(concern+"ProjectNumber", outputs.ProjectNumber)
	}

	if deliveryProjectNumber == "" {
		return nil
	}

	deliverySA := fmt.Sprintf("serviceAccount:%s-compute@developer.gserviceaccount.com", deliveryProjectNumber)
	if err := iambindings.Apply(ctx, []iambindings.Binding{{
		Name:      pc.Name + "-delivery-run-deployer",
		ProjectID: pc.Name,
		Role:      "roles/run.developer",
		Member:    deliverySA,
	}}); err != nil {
		return fmt.Errorf("grant deploy to %s: %w", pc.Name, err)
	}

	return nil
}

// Workloads runs workloads as a standalone Pulumi program.
// Enterprise mode — reads everything from config.
func Workloads() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
