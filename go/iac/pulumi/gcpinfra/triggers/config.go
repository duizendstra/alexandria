package triggers

import "errors"

var (
	// ErrNameRequired means the trigger has no identifier.
	ErrNameRequired = errors.New("triggers: Name is required")
	// ErrTagPatternRequired means the trigger has no Git tag pattern.
	ErrTagPatternRequired = errors.New("triggers: TagPattern is required")
	// ErrConfigFileRequired means the trigger has no build config path.
	ErrConfigFileRequired = errors.New("triggers: ConfigFile is required")
)

// Config defines a Cloud Build trigger.
type Config struct {
	// Name is the trigger identifier.
	Name string `json:"name"`
	// TagPattern is the Git tag regex that fires this trigger.
	TagPattern string `json:"tagPattern"`
	// ConfigFile is the build config path (e.g. "cloudbuild.yaml").
	ConfigFile string `json:"configFile"`
	// Substitutions are key-value pairs passed to the build.
	Substitutions map[string]string `json:"substitutions"`
	// RequireApproval gates the build on manual approval.
	RequireApproval bool `json:"requireApproval"`
}

// Validate checks that the trigger configuration is complete.
func (c Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.TagPattern == "" {
		return ErrTagPatternRequired
	}
	if c.ConfigFile == "" {
		return ErrConfigFileRequired
	}

	return nil
}
