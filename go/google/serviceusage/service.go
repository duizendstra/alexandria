package serviceusage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultMaxAttempts  = 60
)

// Sentinel errors.
var (
	// ErrEmptyProjectID is returned when an operation receives an empty project ID.
	ErrEmptyProjectID = errors.New("project ID cannot be empty")

	// ErrAPIEnablementTimeout is returned when polling for API enablement exceeds maximum attempts.
	ErrAPIEnablementTimeout = errors.New("timeout waiting for API enablement to complete")

	// ErrAPIEnablementFailed is returned when the operation returns an error status.
	ErrAPIEnablementFailed = errors.New("api enablement failed")
)

// Config holds runtime configuration options for Service.
type Config struct {
	Logger       *slog.Logger
	PollInterval time.Duration
	MaxAttempts  int
}

// Service provides Google Cloud Service Usage API operations.
type Service struct {
	usage        *serviceusage.Service
	log          *slog.Logger
	pollInterval time.Duration
	maxAttempts  int
}

// New creates a new Service Usage client instance using provided client options.
func New(ctx context.Context, cfg Config, clientOpts ...option.ClientOption) (*Service, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	usageSvc, err := serviceusage.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("serviceusage: create client: %w", err)
	}

	return &Service{
		usage:        usageSvc,
		log:          log,
		pollInterval: pollInterval,
		maxAttempts:  maxAttempts,
	}, nil
}

// CheckAPIsStatus queries which of the target APIs are currently enabled on the GCP project.
func (s *Service) CheckAPIsStatus(ctx context.Context, projectID string, targetAPIs []string) (map[string]bool, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrEmptyProjectID
	}

	parent := "projects/" + projectID
	enabledMap := make(map[string]bool)

	err := s.usage.Services.List(parent).Filter("state:ENABLED").Context(ctx).Pages(ctx, func(resp *serviceusage.ListServicesResponse) error {
		for _, svc := range resp.Services {
			if svc.Config != nil && svc.Config.Name != "" {
				enabledMap[svc.Config.Name] = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("serviceusage: list enabled services for %s: %w", projectID, err)
	}

	statusMap := make(map[string]bool, len(targetAPIs))
	for _, api := range targetAPIs {
		statusMap[api] = enabledMap[api]
	}

	return statusMap, nil
}

// BatchEnableAPIs initiates batch enablement of the provided service IDs and polls until completion.
func (s *Service) BatchEnableAPIs(ctx context.Context, projectID string, apis []string) error {
	if strings.TrimSpace(projectID) == "" {
		return ErrEmptyProjectID
	}
	if len(apis) == 0 {
		return nil
	}

	parent := "projects/" + projectID
	req := &serviceusage.BatchEnableServicesRequest{
		ServiceIds: apis,
	}

	s.log.Info("triggering batch API enablement",
		slog.String("project_id", projectID),
		slog.Int("api_count", len(apis)))

	op, err := s.usage.Services.BatchEnable(parent, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("serviceusage: batch enable APIs for %s: %w", projectID, err)
	}

	opService := serviceusage.NewOperationsService(s.usage)
	for range s.maxAttempts {
		select {
		case <-ctx.Done():
			return fmt.Errorf("serviceusage: API enablement interrupted: %w", ctx.Err())
		case <-time.After(s.pollInterval):
		}

		statusOp, err := opService.Get(op.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("serviceusage: query operation status %s: %w", op.Name, err)
		}

		if statusOp.Done {
			if statusOp.Error != nil {
				return fmt.Errorf("%w: %s", ErrAPIEnablementFailed, statusOp.Error.Message)
			}

			s.log.Info("batch API enablement complete", slog.String("project_id", projectID))

			return nil
		}
	}

	return fmt.Errorf("%w: operation %s after %d attempts", ErrAPIEnablementTimeout, op.Name, s.maxAttempts)
}
