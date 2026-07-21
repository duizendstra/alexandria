package observability

import (
	"fmt"
	"strings"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/datasets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/logsinks"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/uptimechecks"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/monitoring"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defaultRegion is used when no "region" config is set.
const defaultRegion = "europe-west4"

// defaultGovernanceFolder is the folder key read from the governance
// stack's folder-ID export map when no "governanceFolder" config is set.
const defaultGovernanceFolder = "shared"

// defaultURLOutputKey is the stack-reference output an uptime target reads
// its probed URL from when the target sets no urlOutputKey.
const defaultURLOutputKey = "frontendUrl"

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

	if err := applyMonitoring(ctx, cfg, projectOutputs.ProjectID); err != nil {
		return err
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

// uptimeTarget is one HTTPS endpoint to monitor, read from the optional
// "uptimeTargets" JSON config array. The probed URL comes from the named
// stack reference's URLOutputKey output (default "frontendUrl").
type uptimeTarget struct {
	DisplayName   string   `json:"displayName"`
	StackRef      string   `json:"stackRef"`
	URLOutputKey  string   `json:"urlOutputKey"`
	StatusClasses []string `json:"statusClasses"`
}

// applyMonitoring creates the ops-email notification channel (when the
// "alertEmail" config is set) and, for each configured uptime target, an
// HTTPS uptime check with a failure alert routed to that channel. It is a
// no-op when neither is configured.
func applyMonitoring(ctx *pulumi.Context, cfg *config.Config, projectID pulumi.StringOutput) error {
	var channelIDs pulumi.StringArray
	if email := cfg.Get("alertEmail"); email != "" {
		channel, err := monitoring.NewNotificationChannel(ctx, "ops-email", &monitoring.NotificationChannelArgs{
			Project:     projectID,
			DisplayName: pulumi.Sprintf("Observability ops alerts: %s", email),
			Type:        pulumi.String("email"),
			Labels: pulumi.StringMap{
				"email_address": pulumi.String(email),
			},
		})
		if err != nil {
			return fmt.Errorf("ops email channel: %w", err)
		}
		channelIDs = append(channelIDs, channel.ID())
	}

	targets, err := uptimeTargetsFromConfig(cfg)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := applyUptimeTarget(ctx, projectID, channelIDs, t); err != nil {
			return err
		}
	}

	return nil
}

// uptimeTargetsFromConfig reads and parses the optional "uptimeTargets" JSON
// config array. Absent config yields no targets.
func uptimeTargetsFromConfig(cfg *config.Config) ([]uptimeTarget, error) {
	if cfg.Get("uptimeTargets") == "" {
		return nil, nil
	}

	var targets []uptimeTarget
	if err := cfg.GetObject("uptimeTargets", &targets); err != nil {
		return nil, fmt.Errorf("parse uptimeTargets: %w", err)
	}

	return targets, nil
}

// applyUptimeTarget resolves one target's URL from its stack reference and
// provisions the uptime check + failure alert for it.
func applyUptimeTarget(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	channelIDs pulumi.StringArrayInput,
	t uptimeTarget,
) error {
	ref, err := pulumi.NewStackReference(ctx, t.StackRef, nil)
	if err != nil {
		return fmt.Errorf("uptime target %q stack ref %q: %w", t.DisplayName, t.StackRef, err)
	}

	key := t.URLOutputKey
	if key == "" {
		key = defaultURLOutputKey
	}
	// The URL is a stack-ref output; derive the host with a string transform
	// (not resource creation) so it flows to the check as a Pulumi input.
	//nolint:forcetypeassert // ApplyT on a func returning string always yields a StringOutput.
	host := ref.GetStringOutput(pulumi.String(key)).ApplyT(hostFromURL).(pulumi.StringOutput)

	classes := t.StatusClasses
	if len(classes) == 0 {
		// IAP-fronted endpoints answer unauthenticated probes with a sign-in redirect.
		classes = []string{uptimechecks.Class2xx, uptimechecks.Class3xx}
	}

	if err := uptimechecks.Apply(ctx, projectID, &uptimechecks.Config{
		DisplayName:           t.DisplayName,
		AcceptedStatusClasses: classes,
	}, host, channelIDs, nil); err != nil {
		return fmt.Errorf("uptime check %q: %w", t.DisplayName, err)
	}

	return nil
}

// hostFromURL strips the scheme and any trailing slash from a URL, leaving the
// bare host the uptime check probes.
func hostFromURL(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")

	return strings.TrimSuffix(u, "/")
}

// Observability runs observability as a standalone Pulumi program.
// Enterprise mode — reads everything from config.
func Observability() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
