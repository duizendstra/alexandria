package connections

import "errors"

var (
	// ErrNameRequired means the connection has no identifier.
	ErrNameRequired = errors.New("connections: Name is required")
	// ErrProviderRequired means the connection has no hosting platform.
	ErrProviderRequired = errors.New("connections: Provider is required")
	// ErrRegionRequired means the connection has no deployment region.
	ErrRegionRequired = errors.New("connections: Region is required")
	// ErrRepoLinkNameRequired means the repo link has no identifier.
	ErrRepoLinkNameRequired = errors.New("connections: RepoLink Name is required")
	// ErrRepoLinkRemoteURIRequired means the repo link has no Git remote URL.
	ErrRepoLinkRemoteURIRequired = errors.New("connections: RepoLink RemoteURI is required")
	// ErrUnsupportedProvider means the hosting platform is not implemented.
	ErrUnsupportedProvider = errors.New("connections: unsupported provider")
)

// Config defines a source code connection to a Git hosting provider.
type Config struct {
	// Name is the connection identifier.
	Name string `json:"name"`
	// Provider is the hosting platform ("github", "gitlab", "bitbucket").
	Provider string `json:"provider"`
	// Region is the deployment region.
	Region string `json:"region"`
	// AppInstallationID is the provider app installation (GitHub-specific).
	AppInstallationID int `json:"appInstallationId"`
	// OAuthSecretVersion is the full secret version path for OAuth tokens.
	OAuthSecretVersion string `json:"oauthSecretVersion"`
}

// Validate checks that the connection configuration is complete.
func (c Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Provider == "" {
		return ErrProviderRequired
	}
	if c.Region == "" {
		return ErrRegionRequired
	}

	return nil
}

// RepoLink defines a source repository linked to a connection.
type RepoLink struct {
	// Name is the repository link identifier.
	Name string `json:"name"`
	// RemoteURI is the full Git remote URL.
	RemoteURI string `json:"remoteUri"`
}

// Validate checks that the repo link is complete.
func (r RepoLink) Validate() error {
	if r.Name == "" {
		return ErrRepoLinkNameRequired
	}
	if r.RemoteURI == "" {
		return ErrRepoLinkRemoteURIRequired
	}

	return nil
}
