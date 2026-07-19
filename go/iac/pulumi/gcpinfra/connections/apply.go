package connections

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudbuildv2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Outputs holds references to the created connection.
type Outputs struct {
	ConnectionName pulumi.StringOutput
}

// RepoOutputs holds references to a linked repository.
type RepoOutputs struct {
	RepoID pulumi.IDOutput
}

// Apply creates a Cloud Build v2 connection to a Git provider.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg Config, deps []pulumi.Resource) (*Outputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	args := &cloudbuildv2.ConnectionArgs{
		Project:  projectID,
		Location: pulumi.String(cfg.Region),
		Name:     pulumi.String(cfg.Name),
	}

	switch cfg.Provider {
	case "github":
		ghCfg := &cloudbuildv2.ConnectionGithubConfigArgs{
			AppInstallationId: pulumi.Int(cfg.AppInstallationID),
		}
		if cfg.OAuthSecretVersion != "" {
			ghCfg.AuthorizerCredential = &cloudbuildv2.ConnectionGithubConfigAuthorizerCredentialArgs{
				OauthTokenSecretVersion: pulumi.String(cfg.OAuthSecretVersion),
			}
		}
		args.GithubConfig = ghCfg
	default:
		return nil, fmt.Errorf("%w: %q — supported: github", ErrUnsupportedProvider, cfg.Provider)
	}

	conn, err := cloudbuildv2.NewConnection(ctx, cfg.Name, args, pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("create connection %s: %w", cfg.Name, err)
	}

	return &Outputs{ConnectionName: conn.Name}, nil
}

// LinkRepo creates a repository link under an existing connection.
func LinkRepo(ctx *pulumi.Context, projectID, connName pulumi.StringOutput, region string, repo RepoLink) (*RepoOutputs, error) {
	if err := repo.Validate(); err != nil {
		return nil, err
	}

	r, err := cloudbuildv2.NewRepository(ctx, repo.Name, &cloudbuildv2.RepositoryArgs{
		Project:          projectID,
		Location:         pulumi.String(region),
		Name:             pulumi.String(repo.Name),
		ParentConnection: connName,
		RemoteUri:        pulumi.String(repo.RemoteURI),
	})
	if err != nil {
		return nil, fmt.Errorf("link repo %s: %w", repo.Name, err)
	}

	return &RepoOutputs{RepoID: r.ID()}, nil
}
