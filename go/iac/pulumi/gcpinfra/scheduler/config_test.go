package scheduler_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/scheduler"
)

func TestValidateValid(t *testing.T) {
	c := scheduler.Config{Name: "heartbeat", Region: "europe-west1", Schedule: "*/5 6-22 * * *", TimeZone: "Europe/Amsterdam"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	c := scheduler.Config{Region: "r", Schedule: "s", TimeZone: "tz"}
	if err := c.Validate(); !errors.Is(err, scheduler.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
}

func TestValidateMissingRegion(t *testing.T) {
	c := scheduler.Config{Name: "n", Schedule: "s", TimeZone: "tz"}
	if err := c.Validate(); !errors.Is(err, scheduler.ErrRegionRequired) {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestValidateMissingSchedule(t *testing.T) {
	c := scheduler.Config{Name: "n", Region: "r", TimeZone: "tz"}
	if err := c.Validate(); !errors.Is(err, scheduler.ErrScheduleRequired) {
		t.Errorf("expected ErrScheduleRequired, got %v", err)
	}
}

func TestValidateMissingTimeZone(t *testing.T) {
	c := scheduler.Config{Name: "n", Region: "r", Schedule: "s"}
	if err := c.Validate(); !errors.Is(err, scheduler.ErrTimeZoneRequired) {
		t.Errorf("expected ErrTimeZoneRequired, got %v", err)
	}
}
