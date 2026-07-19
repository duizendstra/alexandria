package registries_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/registries"
)

const formatDocker = "DOCKER"

func TestValidateValid(t *testing.T) {
	c := registries.Config{ID: "example", Format: formatDocker, Location: "europe-west4"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingID(t *testing.T) {
	c := registries.Config{Format: formatDocker, Location: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestValidateMissingFormat(t *testing.T) {
	c := registries.Config{ID: "x", Location: "y"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing Format")
	}
}

func TestValidateMissingLocation(t *testing.T) {
	c := registries.Config{ID: "x", Format: formatDocker}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing Location")
	}
}
