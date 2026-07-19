package cloudrun

import (
	"errors"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// ErrServiceNameRequired means the service has no name.
	ErrServiceNameRequired = errors.New("cloudrun: service Name is required")
	// ErrServiceRegionRequired means the service has no region.
	ErrServiceRegionRequired = errors.New("cloudrun: service Region is required")
	// ErrServiceImageRequired means the service has no container image.
	ErrServiceImageRequired = errors.New("cloudrun: service Image is required")
	// ErrJobNameRequired means the job has no name.
	ErrJobNameRequired = errors.New("cloudrun: job Name is required")
	// ErrJobRegionRequired means the job has no region.
	ErrJobRegionRequired = errors.New("cloudrun: job Region is required")
	// ErrJobImageRequired means the job has no container image.
	ErrJobImageRequired = errors.New("cloudrun: job Image is required")
)

// EnvVar is an environment variable for a Cloud Run container.
// Value supports dynamic pulumi outputs (e.g. from other resources).
type EnvVar struct {
	Name  string
	Value pulumi.StringInput
}

// ServiceConfig defines a Cloud Run service to be provisioned.
type ServiceConfig struct {
	// Name is the Cloud Run service name.
	Name string
	// Region is the GCP region.
	Region string
	// Image is the initial container image.
	Image string
	// Memory is the memory limit for the container (e.g. "512Mi").
	Memory string
}

// Validate checks that the service configuration is complete.
func (c *ServiceConfig) Validate() error {
	if c.Name == "" {
		return ErrServiceNameRequired
	}
	if c.Region == "" {
		return ErrServiceRegionRequired
	}
	if c.Image == "" {
		return ErrServiceImageRequired
	}

	return nil
}

// JobConfig defines a Cloud Run job to be provisioned.
type JobConfig struct {
	// Name is the Cloud Run job name.
	Name string
	// Region is the GCP region.
	Region string
	// Image is the initial container image.
	Image string
	// MaxRetries is the maximum number of retries.
	MaxRetries int
	// Memory is the memory limit for the container (e.g. "512Mi").
	Memory string
}

// Validate checks that the job configuration is complete.
func (c *JobConfig) Validate() error {
	if c.Name == "" {
		return ErrJobNameRequired
	}
	if c.Region == "" {
		return ErrJobRegionRequired
	}
	if c.Image == "" {
		return ErrJobImageRequired
	}

	return nil
}
