package scheduler

import "errors"

var (
	// ErrNameRequired means the scheduler job has no name.
	ErrNameRequired = errors.New("scheduler: Name is required")
	// ErrRegionRequired means the scheduler job has no region.
	ErrRegionRequired = errors.New("scheduler: Region is required")
	// ErrScheduleRequired means the scheduler job has no cron expression.
	ErrScheduleRequired = errors.New("scheduler: Schedule is required")
	// ErrTimeZoneRequired means the scheduler job has no timezone.
	ErrTimeZoneRequired = errors.New("scheduler: TimeZone is required")
)

// Config defines a Cloud Scheduler job to be provisioned.
type Config struct {
	// Name is the scheduler job name.
	Name string
	// Region is the GCP region (e.g. "europe-west1").
	Region string
	// Schedule is a cron expression (e.g. "*/5 6-22 * * *").
	Schedule string
	// TimeZone is the IANA timezone (e.g. "Europe/Amsterdam").
	TimeZone string
	// Paused starts the scheduler in paused state (manual trigger only).
	Paused bool
	// HTTPMethod is the HTTP method (e.g. "POST").
	HTTPMethod string
}

// Validate checks that the configuration is complete.
func (c *Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Region == "" {
		return ErrRegionRequired
	}
	if c.Schedule == "" {
		return ErrScheduleRequired
	}
	if c.TimeZone == "" {
		return ErrTimeZoneRequired
	}

	return nil
}
