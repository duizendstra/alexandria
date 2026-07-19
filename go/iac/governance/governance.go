package governance

import (
	"fmt"
	"sort"

	"github.com/duizendstra/alexandria/go/governance/classification"
	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/governance/plan"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/folders"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/tagkeys"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Apply creates governance resources from Pulumi config.
// Reads config → determines tier → builds Plan → validates → deploys.
func Apply(ctx *pulumi.Context) error {
	cfg := config.New(ctx, "")

	parent := cfg.Require("parent")
	rootFolder := cfg.Require("rootFolder")
	tierName := cfg.Get("tier")

	// Determine scope from GCP-specific parent format.
	s, err := folders.ParseScope(parent)
	if err != nil {
		return fmt.Errorf("governance: %w", err)
	}

	// Organization scope exports the org ID downstream; Container scope
	// must leave it empty (folders.OrgID passes non-org parents through).
	var orgID string
	if s.SupportsOrgExport() {
		orgID = folders.OrgID(parent)
	}

	// Build the plan based on tier.
	var p *plan.Plan

	switch tierName {
	case "starter":
		p, err = plan.NewStarter(s, parent, rootFolder, orgID)

	case "enterprise":
		var children []string
		cfg.RequireObject("environments", &children)

		var dims []classification.Dimension
		_ = cfg.TryObject("tagKeys", &dims)

		p, err = plan.NewEnterprise(parent, rootFolder, children, dims,
			cfg.Get("billingAccount"), orgID)

	default: // "standard" or unset — backward compatible default.
		var children []string
		cfg.RequireObject("environments", &children)

		p, err = plan.NewStandard(s, parent, rootFolder, children, orgID)
	}

	if err != nil {
		return fmt.Errorf("governance: %w", err)
	}

	// --- Deploy (adapter calls) ---.

	// 1. Folder hierarchy.
	folderOutputs, err := folders.Apply(ctx, p.Hierarchy)
	if err != nil {
		return fmt.Errorf("governance: %w", err)
	}

	// 2. Classification dimensions — Enterprise only.
	if p.Scope.SupportsClassification() && len(p.Dimensions) > 0 {
		tagOutputs, tErr := tagkeys.Apply(ctx, p.OrgID, p.Dimensions)
		if tErr != nil {
			return fmt.Errorf("governance: %w", tErr)
		}

		tagNames := make([]string, 0, len(tagOutputs))
		for name := range tagOutputs {
			tagNames = append(tagNames, name)
		}

		sort.Strings(tagNames)

		for _, name := range tagNames {
			ctx.Export(exports.TagKeyID(name), tagOutputs[name])
		}
	}

	// --- Exports (downstream BC contract) ---.

	if p.BillingAccount != "" {
		ctx.Export(exports.BillingAccount, pulumi.String(p.BillingAccount))
	}

	ctx.Export(exports.RootFolderID, folderOutputs.RootFolderID)
	ctx.Export(exports.FolderIDs, folderOutputs.FolderIDs)

	if p.Scope.SupportsOrgExport() {
		ctx.Export(exports.OrgID, pulumi.String(p.OrgID))
	}

	return nil
}

// Governance runs governance as a standalone Pulumi program.
func Governance() {
	pulumi.Run(Apply)
}
