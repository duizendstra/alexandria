package delivery

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// optionalObject unmarshals the optional config key into out. An absent
// key is fine and leaves out untouched; a key that is present but does
// not unmarshal is fatal. Reading a malformed block as empty would let
// the next update delete every resource the block previously declared.
func optionalObject(cfg *config.Config, key string, out any) error {
	err := cfg.TryObject(key, out)
	if err == nil || errors.Is(err, config.ErrMissingVar) {
		return nil
	}

	return fmt.Errorf("config key %q: %w", key, err)
}
