package delivery

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/connections"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/registries"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/triggers"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defaultRegion is used when no "region" config is set.
const defaultRegion = "europe-west4"

// defaultGovernanceFolder is the folder key read from the governance
// stack's folder-ID export map when no "governanceFolder" config is set.
const defaultGovernanceFolder = "shared"

// defaultConnectionName is used when no "githubConnectionName" config is set.
const defaultConnectionName = "github"

// RepoConfig defines a source repository and its build triggers.
type RepoConfig struct {
	Name      string          `json:"name"`
	RemoteURI string          `json:"remoteURI"`
	Triggers  []TriggerConfig `json:"triggers"`
}

// TriggerConfig defines a Cloud Build trigger for a repository.
type TriggerConfig struct {
	Name            string            `json:"name"`
	TagPattern      string            `json:"tagPattern"`
	ConfigFile      string            `json:"configFile"`
	RequireApproval bool              `json:"requireApproval"`
	Substitutions   map[string]string `json:"substitutions"`
}

// Params allows upstream BCs to pass values directly (collapsed mode).
// When nil, all values come from Pulumi stack config (enterprise mode).
type Params struct {
	// FolderID overrides config — used when governance passes it directly.
	FolderID string
	// BillingAccount overrides config.
	BillingAccount string
}

// placement is where the delivery project lands and who pays for it.
type placement struct {
	folderID       string
	billingAccount string
}

// Apply creates the delivery resources: project, registry, Git
// connection, triggers, and consumer reader grants.
//
// Resolution order for folderID and billingAccount:
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
			"cloudbuild.googleapis.com",
			"artifactregistry.googleapis.com",
			"secretmanager.googleapis.com",
			"iam.googleapis.com",
			"compute.googleapis.com",
		},
	})
	if err != nil {
		return fmt.Errorf("delivery project: %w", err)
	}

	arOutputs, err := applyRegistry(ctx, cfg, region, projectOutputs)
	if err != nil {
		return err
	}

	if done, gErr := applyGitHub(ctx, cfg, region, projectOutputs, arOutputs); gErr != nil {
		return gErr
	} else if done {
		return nil
	}

	if err := applyConsumerGrants(ctx, cfg, region, projectOutputs, arOutputs); err != nil {
		return err
	}

	ctx.Export("projectId", projectOutputs.ProjectID)
	ctx.Export("dockerRepoId", arOutputs.RepositoryID)

	return nil
}

// resolvePlacement determines folder ID and billing account from params,
// the governance stack reference, and stack config, in that order.
func resolvePlacement(ctx *pulumi.Context, cfg *config.Config, params *Params) placement {
	place := placement{}
	if params != nil {
		place.folderID = params.FolderID
		place.billingAccount = params.BillingAccount
	}

	if place.folderID == "" || place.billingAccount == "" {
		fromGovernanceStack(ctx, cfg, &place)
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

// applyRegistry creates the container registry and grants the Cloud
// Build default SA write access to it.
func applyRegistry(ctx *pulumi.Context, cfg *config.Config, region string, projectOutputs *projects.Outputs) (*registries.Outputs, error) {
	arID := cfg.Require("registryId")

	arDesc := cfg.Get("registryDescription")
	if arDesc == "" {
		arDesc = "Container images"
	}

	arOutputs, err := registries.Apply(ctx, projectOutputs.ProjectID, registries.Config{
		ID:          arID,
		Description: arDesc,
		Format:      "DOCKER",
		Location:    region,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}

	buildSA := pulumi.Sprintf("serviceAccount:%s@cloudbuild.gserviceaccount.com", projectOutputs.ProjectNumber)

	if err := registries.GrantWriter(ctx, "build-ar-writer", projectOutputs.ProjectID, region, arOutputs.Name, buildSA); err != nil {
		return nil, fmt.Errorf("build writer grant: %w", err)
	}

	return arOutputs, nil
}

// applyGitHub creates the Git hosting connection and the configured
// repository links and triggers. When the connection is not yet
// configured it exports guidance and reports done=true so Apply can
// stop early (matching the operator-guided first deploy).
func applyGitHub(ctx *pulumi.Context, cfg *config.Config, region string, projectOutputs *projects.Outputs, arOutputs *registries.Outputs) (bool, error) {
	connectionName := cfg.Get("githubConnectionName")
	if connectionName == "" {
		connectionName = defaultConnectionName
	}

	appID := cfg.GetInt("githubAppInstallationId")
	oauthSecret := cfg.Get("githubOAuthSecretVersion")

	if appID == 0 || oauthSecret == "" {
		// Guide the operator through the manual setup steps.
		ctx.Export("nextStep", pulumi.String("GitHub connection not configured — set githubAppInstallationId and githubOAuthSecretVersion"))
		ctx.Export("projectId", projectOutputs.ProjectID)
		ctx.Export("dockerRepoId", arOutputs.RepositoryID)

		return true, nil
	}

	connOutputs, err := connections.Apply(ctx, projectOutputs.ProjectID, connections.Config{
		Name:               connectionName,
		Provider:           "github",
		Region:             region,
		AppInstallationID:  appID,
		OAuthSecretVersion: oauthSecret,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("github connection: %w", err)
	}

	return false, applyRepos(ctx, cfg, region, projectOutputs, connOutputs)
}

// applyRepos links the configured repositories and creates their
// build triggers.
func applyRepos(ctx *pulumi.Context, cfg *config.Config, region string, projectOutputs *projects.Outputs, connOutputs *connections.Outputs) error {
	var repos []RepoConfig
	_ = cfg.TryObject("repositories", &repos)

	triggerSA := pulumi.Sprintf("projects/%s/serviceAccounts/%s-compute@developer.gserviceaccount.com",
		projectOutputs.ProjectID, projectOutputs.ProjectNumber)

	for _, repo := range repos {
		repoLink, err := connections.LinkRepo(ctx, projectOutputs.ProjectID, connOutputs.ConnectionName, region, connections.RepoLink{
			Name:      repo.Name,
			RemoteURI: repo.RemoteURI,
		})
		if err != nil {
			return fmt.Errorf("link repo %s: %w", repo.Name, err)
		}

		for _, t := range repo.Triggers {
			if err := triggers.Apply(ctx, projectOutputs.ProjectID, region, repoLink.RepoID, triggerSA, triggers.Config{
				Name:            t.Name,
				TagPattern:      t.TagPattern,
				ConfigFile:      t.ConfigFile,
				RequireApproval: t.RequireApproval,
				Substitutions:   t.Substitutions,
			}, nil); err != nil {
				return fmt.Errorf("trigger %s: %w", t.Name, err)
			}
		}
	}

	return nil
}

// applyConsumerGrants grants the Cloud Run service agent of each
// consumer workload stack reader access to the registry so it can pull
// images. Each workload stack exports {concern}ProjectNumber; the
// compute concern's number is read from every configured stack.
func applyConsumerGrants(ctx *pulumi.Context, cfg *config.Config, region string, projectOutputs *projects.Outputs, arOutputs *registries.Outputs) error {
	var consumerWorkloadStacks []string
	_ = cfg.TryObject("consumerWorkloadStacks", &consumerWorkloadStacks)

	for i, stackRef := range consumerWorkloadStacks {
		ref, err := pulumi.NewStackReference(ctx, stackRef, nil)
		if err != nil {
			return fmt.Errorf("workload stack ref %s: %w", stackRef, err)
		}

		projectNumber := ref.GetStringOutput(pulumi.String("computeProjectNumber"))
		member := pulumi.Sprintf("serviceAccount:service-%s@serverless-robot-prod.iam.gserviceaccount.com", projectNumber)

		if err := registries.GrantReader(ctx, fmt.Sprintf("ar-consumer-%d", i), projectOutputs.ProjectID, region, arOutputs.Name, member); err != nil {
			return fmt.Errorf("consumer grant %d: %w", i, err)
		}
	}

	return nil
}

// Delivery runs delivery as a standalone Pulumi program.
// Enterprise mode — reads everything from config.
func Delivery() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
