package cloudrun_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/cloudrun"
)

const regionTest = "europe-west4"

func TestServiceValidateValid(t *testing.T) {
	c := cloudrun.ServiceConfig{Name: "api", Region: regionTest, Image: "gcr.io/example/api:latest"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServiceValidateMissingName(t *testing.T) {
	c := cloudrun.ServiceConfig{Region: "r", Image: "i"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrServiceNameRequired) {
		t.Errorf("expected ErrServiceNameRequired, got %v", err)
	}
}

func TestServiceValidateMissingRegion(t *testing.T) {
	c := cloudrun.ServiceConfig{Name: "n", Image: "i"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrServiceRegionRequired) {
		t.Errorf("expected ErrServiceRegionRequired, got %v", err)
	}
}

func TestServiceValidateMissingImage(t *testing.T) {
	c := cloudrun.ServiceConfig{Name: "n", Region: "r"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrServiceImageRequired) {
		t.Errorf("expected ErrServiceImageRequired, got %v", err)
	}
}

func TestJobValidateValid(t *testing.T) {
	c := cloudrun.JobConfig{Name: "worker", Region: regionTest, Image: "gcr.io/example/worker:latest"}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJobValidateMissingName(t *testing.T) {
	c := cloudrun.JobConfig{Region: "r", Image: "i"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrJobNameRequired) {
		t.Errorf("expected ErrJobNameRequired, got %v", err)
	}
}

func TestJobValidateMissingRegion(t *testing.T) {
	c := cloudrun.JobConfig{Name: "n", Image: "i"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrJobRegionRequired) {
		t.Errorf("expected ErrJobRegionRequired, got %v", err)
	}
}

func TestJobValidateMissingImage(t *testing.T) {
	c := cloudrun.JobConfig{Name: "n", Region: "r"}
	if err := c.Validate(); !errors.Is(err, cloudrun.ErrJobImageRequired) {
		t.Errorf("expected ErrJobImageRequired, got %v", err)
	}
}
