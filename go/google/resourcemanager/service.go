package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultMaxAttempts  = 30
)

// Sentinel errors.
var (
	// ErrEmptyProjectID is returned when an operation receives an empty project ID.
	ErrEmptyProjectID = errors.New("project ID cannot be empty")

	// ErrEmptyBillingAccountID is returned when a billing operation receives an empty billing account ID.
	ErrEmptyBillingAccountID = errors.New("billing account ID cannot be empty")

	// ErrProjectCreationTimeout is returned when polling for project activation exceeds maximum attempts.
	ErrProjectCreationTimeout = errors.New("timeout waiting for project to become active")
)

// ProjectRequest holds parameters for creating a new GCP Project.
type ProjectRequest struct {
	ProjectID string
	Name      string
	FolderID  string
	OrgID     string
	Labels    map[string]string
}

// Config holds runtime configuration options for Service.
type Config struct {
	Logger       *slog.Logger
	PollInterval time.Duration
	MaxAttempts  int
}

// Service provides GCP Project lifecycle and Cloud Billing operations.
type Service struct {
	crm          *cloudresourcemanager.Service
	billing      *cloudbilling.APIService
	log          *slog.Logger
	pollInterval time.Duration
	maxAttempts  int
}

// New creates a new Resource Manager Service instance using provided client options.
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

	crmSvc, err := cloudresourcemanager.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("resourcemanager: create CRM client: %w", err)
	}

	billingSvc, err := cloudbilling.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("resourcemanager: create billing client: %w", err)
	}

	return &Service{
		crm:          crmSvc,
		billing:      billingSvc,
		log:          log,
		pollInterval: pollInterval,
		maxAttempts:  maxAttempts,
	}, nil
}

// CheckProjectExists queries the Cloud Resource Manager API to verify if a project exists.
func (s *Service) CheckProjectExists(ctx context.Context, projectID string) (bool, error) {
	if strings.TrimSpace(projectID) == "" {
		return false, ErrEmptyProjectID
	}

	_, err := s.crm.Projects.Get(projectID).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("resourcemanager: get project %s: %w", projectID, err)
	}

	return true, nil
}

// CreateProject provisions a new GCP Project asynchronously and polls until it is active.
func (s *Service) CreateProject(ctx context.Context, req ProjectRequest) error {
	if strings.TrimSpace(req.ProjectID) == "" {
		return ErrEmptyProjectID
	}

	name := req.Name
	if name == "" {
		name = req.ProjectID
	}

	project := &cloudresourcemanager.Project{
		ProjectId: req.ProjectID,
		Name:      name,
		Labels:    req.Labels,
	}

	if req.FolderID != "" {
		project.Parent = &cloudresourcemanager.ResourceId{
			Id:   req.FolderID,
			Type: "folder",
		}
	} else if req.OrgID != "" {
		project.Parent = &cloudresourcemanager.ResourceId{
			Id:   req.OrgID,
			Type: "organization",
		}
	}

	s.log.Info("initiating GCP project creation",
		slog.String("project_id", req.ProjectID),
		slog.String("parent_folder", req.FolderID),
		slog.String("parent_org", req.OrgID))

	_, err := s.crm.Projects.Create(project).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("resourcemanager: create project %s: %w", req.ProjectID, err)
	}

	for range s.maxAttempts {
		select {
		case <-ctx.Done():
			return fmt.Errorf("resourcemanager: project creation interrupted: %w", ctx.Err())
		case <-time.After(s.pollInterval):
		}

		exists, checkErr := s.CheckProjectExists(ctx, req.ProjectID)
		if checkErr == nil && exists {
			s.log.Info("GCP project active", slog.String("project_id", req.ProjectID))

			return nil
		}
	}

	return fmt.Errorf("%w: project %s after %d attempts", ErrProjectCreationTimeout, req.ProjectID, s.maxAttempts)
}

// CheckBillingLinked checks if a specific billing account is actively associated with the project.
func (s *Service) CheckBillingLinked(ctx context.Context, projectID, billingAccountID string) (bool, error) {
	if strings.TrimSpace(projectID) == "" {
		return false, ErrEmptyProjectID
	}
	if strings.TrimSpace(billingAccountID) == "" {
		return false, ErrEmptyBillingAccountID
	}

	resourceName := "projects/" + projectID
	info, err := s.billing.Projects.GetBillingInfo(resourceName).Context(ctx).Do()
	if err != nil {
		return false, fmt.Errorf("resourcemanager: query billing info for %s: %w", projectID, err)
	}

	targetName := "billingAccounts/" + billingAccountID
	linked := info.BillingEnabled && info.BillingAccountName == targetName

	return linked, nil
}

// LinkBilling links a billing account to a GCP project.
func (s *Service) LinkBilling(ctx context.Context, projectID, billingAccountID string) error {
	if strings.TrimSpace(projectID) == "" {
		return ErrEmptyProjectID
	}
	if strings.TrimSpace(billingAccountID) == "" {
		return ErrEmptyBillingAccountID
	}

	resourceName := "projects/" + projectID
	targetAccount := "billingAccounts/" + billingAccountID

	s.log.Info("linking billing account to project",
		slog.String("project_id", projectID),
		slog.String("billing_account_id", billingAccountID))

	_, err := s.billing.Projects.UpdateBillingInfo(resourceName, &cloudbilling.ProjectBillingInfo{
		BillingAccountName: targetAccount,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("resourcemanager: link billing %s to %s: %w", billingAccountID, projectID, err)
	}

	return nil
}

func isNotFound(err error) bool {
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		return gErr.Code == http.StatusNotFound
	}

	return false
}
