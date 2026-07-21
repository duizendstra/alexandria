package uptimechecks_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/uptimechecks"
)

func validConfig() uptimechecks.Config {
	return uptimechecks.Config{
		DisplayName:           "app prod",
		Host:                  "app.example.com",
		Path:                  "/healthz",
		Port:                  443,
		AcceptedStatusClasses: []string{uptimechecks.Class2xx, uptimechecks.Class3xx},
		Period:                "60s",
		Timeout:               "10s",
	}
}

func TestValidateValid(t *testing.T) {
	c := validConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMinimal(t *testing.T) {
	// Only the two required fields set; everything else defaults.
	c := uptimechecks.Config{DisplayName: "x", Host: "h"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error for minimal config: %v", err)
	}
}

func TestValidateMissingDisplayName(t *testing.T) {
	c := validConfig()
	c.DisplayName = ""
	if err := c.Validate(); !errors.Is(err, uptimechecks.ErrDisplayNameRequired) {
		t.Errorf("expected ErrDisplayNameRequired, got %v", err)
	}
}

func TestValidateMissingHost(t *testing.T) {
	c := validConfig()
	c.Host = ""
	if err := c.Validate(); !errors.Is(err, uptimechecks.ErrHostRequired) {
		t.Errorf("expected ErrHostRequired, got %v", err)
	}
}

func TestValidatePortOutOfRange(t *testing.T) {
	c := validConfig()
	c.Port = 70000
	if err := c.Validate(); !errors.Is(err, uptimechecks.ErrPortOutOfRange) {
		t.Errorf("expected ErrPortOutOfRange, got %v", err)
	}
}

func TestValidateBadPeriod(t *testing.T) {
	c := validConfig()
	c.Period = "45s"
	if err := c.Validate(); !errors.Is(err, uptimechecks.ErrPeriodInvalid) {
		t.Errorf("expected ErrPeriodInvalid, got %v", err)
	}
}

func TestValidateBadStatusClass(t *testing.T) {
	c := validConfig()
	c.AcceptedStatusClasses = []string{uptimechecks.Class2xx, "teapot"}
	if err := c.Validate(); !errors.Is(err, uptimechecks.ErrStatusClassInvalid) {
		t.Errorf("expected ErrStatusClassInvalid, got %v", err)
	}
}
