package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/samber/lo"
)

// Instance represents an instance for internal use (domain layer)
type Instance struct {
	ID         string
	UserID     string
	Status     string
	Namespace  string
	Subdomain  string
	AppVersion string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeployedAt *time.Time
	DeletedAt  *time.Time
}

func (i *Instance) GetInstanceURL() string {
	return InstanceURL(i.Subdomain)
}

func InstanceURL(subdomain string) string {
	return fmt.Sprintf("https://%s.ranx.cloud", subdomain)
}

// toDomainInstance maps a db.Instance to a types.Instance (domain layer)
func toDomainInstance(dbInst db.Instance) Instance {
	i := Instance{
		ID:         dbInst.ID,
		UserID:     dbInst.UserID,
		Status:     dbInst.Status,
		Namespace:  dbInst.Namespace,
		Subdomain:  dbInst.Subdomain,
		AppVersion: dbInst.AppVersion,
		CreatedAt:  dbInst.CreatedAt.Time,
		UpdatedAt:  dbInst.UpdatedAt.Time,
	}
	if dbInst.DeployedAt.Valid {
		i.DeployedAt = &dbInst.DeployedAt.Time
	}
	if dbInst.DeletedAt.Valid {
		i.DeletedAt = &dbInst.DeletedAt.Time
	}
	return i
}

func (s *Service) GetInstancesByUser(ctx context.Context, userID string) ([]Instance, error) {

	queries := s.getDB()

	dbInstances, err := queries.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return lo.Map(dbInstances, func(inst db.Instance, _ int) Instance {
		return toDomainInstance(inst)
	}), nil
}

func (s *Service) GetInstanceByID(ctx context.Context, instanceID string) (*Instance, error) {
	queries := s.getDB()

	dbInstance, err := queries.GetInstance(ctx, instanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

func (s *Service) GetInstanceBySubdomain(ctx context.Context, subdomain string) (*Instance, error) {
	queries := s.getDB()

	dbInstance, err := queries.GetInstanceBySubdomain(ctx, subdomain)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

type UpdateInstanceStatusParams struct {
	InstanceID string
	Status     string
}

func (s *Service) UpdateInstanceStatus(ctx context.Context, params UpdateInstanceStatusParams) error {
	queries := s.getDB()

	_, err := queries.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{
		ID:     params.InstanceID,
		Status: params.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}
	return nil
}

func (s *Service) CheckSubdomainExists(ctx context.Context, subdomain string) (bool, error) {
	queries := s.getDB()

	exists, err := queries.CheckSubdomainExists(ctx, subdomain)
	if err != nil {
		return false, fmt.Errorf("failed to check subdomain existence: %w", err)
	}

	return exists, nil
}

func (s *Service) CheckInstanceURLActive(ctx context.Context, id string) (bool, error) {

	queries := s.getDB()

	instance, err := queries.GetInstance(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to get instance: %w", err)
	}

	// if instance created less than 1 minute ago, skip check
	if time.Since(instance.CreatedAt.Time) < time.Minute {
		return false, nil
	}

	healthURL := fmt.Sprintf("https://%s.ranx.cloud/healthz/readiness", instance.Subdomain)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make GET request to instance URL
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		// check if context deadline exceeded
		if ctx.Err() == context.DeadlineExceeded {
			return false, nil
		}
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Instance not ready yet
		return false, nil
	}
	defer resp.Body.Close()

	// Check if we got a 200 status
	return resp.StatusCode == http.StatusOK, nil
}
