package identity

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/duizendstra/alexandria/go/governance/exports"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/iambindings"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/projects"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/secrets"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/serviceaccounts"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defaultGovernanceFolder is the folder key read from the governance
// stack's folder-ID export map when no "governanceFolder" config is set.
const defaultGovernanceFolder = "shared"

// SecretResolver resolves a secret reference to its value.
// The blueprint does not know HOW secrets are sourced — only that
// a resolver provides them. Adapters: PassResolver (default), env vars, vault.
type SecretResolver func(ref string) (string, error)

// PassResolver reads secrets from the local pass store.
// Default for operators who use pass(1).
func PassResolver(ref string) (string, error) {
	out, err := exec.CommandContext(context.Background(), "pass", "show", ref).Output() //nolint:gosec // ref from config
	if err != nil {
		return "", fmt.Errorf("pass show %s: %w", ref, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// Params allows upstream BCs to pass values directly (collapsed mode).
// When nil, all values come from Pulumi stack config (enterprise mode).
type Params struct {
	// FolderID overrides config — used when governance passes it directly.
	FolderID string
	// BillingAccount overrides config.
	BillingAccount string
	// Resolver overrides the default PassResolver.
	Resolver SecretResolver
}

// placement is where the identity project lands and who pays for it.
type placement struct {
	folderID       string
	billingAccount string
}

// Apply creates identity resources: project, secrets, service accounts, IAM.
//
// Resolution order for folderID and billingAccount:
//  1. Params (collapsed mode — governance passes directly)
//  2. Governance stack reference named by the "governanceStack" config key
//  3. Stack config (explicit override / fallback)
func Apply(ctx *pulumi.Context, params *Params) error {
	cfg := config.New(ctx, "")

	projectName := cfg.Require("projectName")

	resolver := SecretResolver(PassResolver)
	if params != nil && params.Resolver != nil {
		resolver = params.Resolver
	}

	place := resolvePlacement(ctx, cfg, params)

	projectOutputs, err := projects.Apply(ctx, projects.Config{
		Name:           projectName,
		FolderID:       place.folderID,
		BillingAccount: place.billingAccount,
		APIs:           []string{"secretmanager.googleapis.com", "iam.googleapis.com"},
	})
	if err != nil {
		return fmt.Errorf("identity project: %w", err)
	}

	if err := applySecrets(ctx, cfg, projectOutputs.ProjectID, resolver); err != nil {
		return err
	}

	if err := applyConsumerAccess(ctx, cfg, projectName); err != nil {
		return err
	}

	if err := applyServiceAccounts(ctx, cfg, projectOutputs.ProjectID, projectName); err != nil {
		return err
	}

	ctx.Export("projectId", projectOutputs.ProjectID)

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

// applySecrets creates the configured secrets, resolving each value at
// deploy time. No-op when no secrets are configured.
func applySecrets(ctx *pulumi.Context, cfg *config.Config, projectID pulumi.StringOutput, resolver SecretResolver) error {
	var secretDefs []struct {
		Name string `json:"name"`
		Ref  string `json:"ref"`
	}
	_ = cfg.TryObject("secrets", &secretDefs)

	if len(secretDefs) == 0 {
		return nil
	}

	ss := make([]secrets.Secret, len(secretDefs))
	for i, s := range secretDefs {
		val, rErr := resolver(s.Ref)
		if rErr != nil {
			return fmt.Errorf("resolve secret %s: %w", s.Name, rErr)
		}
		ss[i] = secrets.Secret{Name: s.Name, Value: val}
	}

	if err := secrets.Apply(ctx, projectID, ss, nil); err != nil {
		return fmt.Errorf("identity secrets: %w", err)
	}

	return nil
}

// applyConsumerAccess grants the configured consumer service accounts
// the Secret Manager accessor role on the identity project.
func applyConsumerAccess(ctx *pulumi.Context, cfg *config.Config, projectName string) error {
	var consumerSAs []string
	cfg.RequireObject("consumerSAs", &consumerSAs)

	bindings := make([]iambindings.Binding, len(consumerSAs))
	for i, sa := range consumerSAs {
		bindings[i] = iambindings.Binding{
			Name:      fmt.Sprintf("consumer-accessor-%d", i),
			ProjectID: projectName,
			Role:      "roles/secretmanager.secretAccessor",
			Member:    "serviceAccount:" + sa,
		}
	}

	if err := iambindings.Apply(ctx, bindings); err != nil {
		return fmt.Errorf("consumer access: %w", err)
	}

	return nil
}

// applyServiceAccounts creates the configured service accounts, exports
// their emails, and grants impersonators access. No-op when no service
// accounts are configured.
func applyServiceAccounts(ctx *pulumi.Context, cfg *config.Config, projectID pulumi.StringOutput, projectName string) error {
	var accountDefs []serviceaccounts.Account
	_ = cfg.TryObject("serviceAccounts", &accountDefs)

	if len(accountDefs) == 0 {
		return nil
	}

	saOutputs, err := serviceaccounts.Apply(ctx, projectID, accountDefs, nil)
	if err != nil {
		return fmt.Errorf("identity service accounts: %w", err)
	}

	for id, email := range saOutputs {
		ctx.Export(id+"-email", email)
	}

	return applyImpersonators(ctx, cfg, projectName, accountDefs)
}

// applyImpersonators grants each configured impersonator the user,
// token-creator, and OIDC-token-creator roles on every identity service
// account. Needed when SAs are used cross-project (e.g. Cloud Run,
// Scheduler). No-op when no impersonators are configured.
func applyImpersonators(ctx *pulumi.Context, cfg *config.Config, projectName string, accountDefs []serviceaccounts.Account) error {
	var impersonators []string
	_ = cfg.TryObject("impersonators", &impersonators)

	if len(impersonators) == 0 {
		return nil
	}

	roles := []struct {
		suffix string
		role   string
	}{
		{suffix: "user", role: "roles/iam.serviceAccountUser"},
		{suffix: "token", role: "roles/iam.serviceAccountTokenCreator"},
		{suffix: "oidc", role: "roles/iam.serviceAccountOpenIdTokenCreator"},
	}

	var saIAMBindings []iambindings.SAIamBinding
	for i, imp := range impersonators {
		for _, acct := range accountDefs {
			for _, r := range roles {
				saIAMBindings = append(saIAMBindings, iambindings.SAIamBinding{
					Name:             fmt.Sprintf("impersonate-%s-%d-%s", acct.ID, i, r.suffix),
					ServiceAccountID: pulumi.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectName, acct.ID, projectName),
					Role:             r.role,
					Member:           pulumi.String(imp),
				})
			}
		}
	}

	if _, err := iambindings.ApplySAIam(ctx, saIAMBindings); err != nil {
		return fmt.Errorf("impersonator bindings: %w", err)
	}

	return nil
}

// Identity runs identity as a standalone Pulumi program.
// Enterprise mode — reads everything from config, uses PassResolver.
func Identity() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	})
}
