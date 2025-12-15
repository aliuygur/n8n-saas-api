package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/samber/lo"
)

// Instance represents an instance for internal use (domain layer)
type Instance struct {
	ID         string
	UserID     string
	Status     string
	Namespace  string
	Subdomain  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeployedAt *time.Time
	DeletedAt  *time.Time
}

func (i *Instance) GetInstanceURL() string {
	return fmt.Sprintf("https://%s.instol.cloud", i.Subdomain)
}

// toDomainInstance maps a db.Instance to a types.Instance (domain layer)
func toDomainInstance(dbInst db.Instance) Instance {
	i := Instance{
		ID:        dbInst.ID,
		UserID:    dbInst.UserID,
		Status:    dbInst.Status,
		Namespace: dbInst.Namespace,
		Subdomain: dbInst.Subdomain,
		CreatedAt: dbInst.CreatedAt.Time,
		UpdatedAt: dbInst.UpdatedAt.Time,
	}
	if dbInst.DeployedAt.Valid {
		i.DeployedAt = &dbInst.DeployedAt.Time
	}
	if dbInst.DeletedAt.Valid {
		i.DeletedAt = &dbInst.DeletedAt.Time
	}
	return i
}

// errors

type InstanceNotFoundError struct{}

func (s *Service) GetInstancesByUser(ctx context.Context, userID string) ([]Instance, error) {

	queries := db.New(s.db)

	dbInstances, err := queries.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return lo.Map(dbInstances, func(inst db.Instance, _ int) Instance {
		return toDomainInstance(inst)
	}), nil
}

func (s *Service) GetInstanceByID(ctx context.Context, instanceID string) (*Instance, error) {
	queries := db.New(s.db)

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

type DeleteInstanceParams struct {
	UserID     string
	InstanceID string
}

func (s *Service) DeleteInstance(ctx context.Context, params DeleteInstanceParams) error {

	if err := s.RunInTransaction(ctx, func(tx *db.Queries) error {
		instance, err := tx.GetInstance(ctx, params.InstanceID)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		if err := s.gke.DeleteNamespace(ctx, instance.Namespace); err != nil {
			return apperrs.Server("failed to delete namespace from Kubernetes", err)
		}
		if err := s.cloudflare.DeleteDNSRecord(ctx, instance.Subdomain); err != nil {
			return apperrs.Server("failed to delete DNS record from Cloudflare", err)
		}

		// Soft delete from database
		if err := tx.DeleteInstance(ctx, params.InstanceID); err != nil {
			return apperrs.Server("failed to delete instance from database", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type UpdateInstanceStatusParams struct {
	InstanceID string
	Status     string
}

func (s *Service) UpdateInstanceStatus(ctx context.Context, params UpdateInstanceStatusParams) error {
	queries := db.New(s.db)

	_, err := queries.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{
		ID:     params.InstanceID,
		Status: params.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}
	return nil
}

type CreateInstanceParams struct {
	UserID    string
	Subdomain string
}

func (s *Service) CreateInstance(ctx context.Context, params CreateInstanceParams) (*Instance, error) {
	if err := domainutils.ValidateSubdomain(params.Subdomain); err != nil {
		return nil, apperrs.Client(apperrs.CodeInvalidSubdomain, "invalid subdomain")
	}

	var dbInstance db.Instance
	if err := s.RunInTransaction(ctx, func(tx *db.Queries) error {

		// one customer can have only one instance with trial status
		instances, err := tx.ListInstancesByUser(ctx, params.UserID)
		if err != nil {
			return apperrs.Server("failed to list user instances", err)
		}

		if len(instances) > 0 {
			return apperrs.Client(apperrs.CodeInvalidInput, "user already has an instance")
		}

		// Check if subdomain already exists
		exists, err := tx.CheckSubdomainExists(ctx, params.Subdomain)
		if err != nil {
			return apperrs.Server("failed to check subdomain existence", err)
		}
		if exists {
			return apperrs.Client(apperrs.CodeConflict, "subdomain already taken")
		}

		namespace, err := s.generateUniqueNamespace(ctx, tx)
		if err != nil {
			return err
		}

		dbInstance, err = tx.CreateInstance(ctx, db.CreateInstanceParams{
			UserID:    params.UserID,
			Namespace: namespace,
			Subdomain: params.Subdomain,
		})
		if err != nil {
			return apperrs.Server("failed to create instance in database", err)
		}

		// Deploy n8n instance
		domain := fmt.Sprintf("https://%s.instol.cloud", params.Subdomain)
		n8nInstance := &n8ntemplates.N8N_V1{
			Namespace:     namespace,
			EncryptionKey: lo.RandomString(32, lo.AllCharset),
			BaseURL:       domain,
		}

		if err := s.gke.Apply(ctx, n8nInstance); err != nil {
			return fmt.Errorf("failed to deploy n8n: %w", err)
		}

		// Add Cloudflare tunnel route for external access
		serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", namespace)
		if err := s.cloudflare.AddTunnelRoute(ctx, domain, serviceURL); err != nil {
			return apperrs.Server("failed to add Cloudflare tunnel route", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// TODO: deploy instance to Kubernetes and create Cloudflare DNS record here

	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

func (s *Service) CheckSubdomainExists(ctx context.Context, subdomain string) (bool, error) {
	queries := db.New(s.db)

	exists, err := queries.CheckSubdomainExists(ctx, subdomain)
	if err != nil {
		return false, fmt.Errorf("failed to check subdomain existence: %w", err)
	}

	return exists, nil
}

func (s *Service) generateUniqueNamespace(ctx context.Context, queries *db.Queries) (string, error) {
	// Try to find a unique namespace
	maxAttempts := 10
	for range maxAttempts {
		// Generate 8-character alphanumeric random string
		randomStr := lo.RandomString(16, append(lo.LowerCaseLettersCharset, lo.NumbersCharset...))

		// Format: n8n-{8-alphanumeric}
		namespace := fmt.Sprintf("n8n-%s", strings.ToLower(randomStr))
		// Truncate if too long (Kubernetes limit is 63 characters)
		if len(namespace) > 63 {
			namespace = namespace[:63]
		}

		// Check if namespace exists in database
		exists, err := queries.CheckNamespaceExists(ctx, namespace)
		if err != nil {
			return "", apperrs.Server("failed to check namespace existence", err)
		}

		if !exists {
			return namespace, nil
		}
	}

	return "", apperrs.Server(fmt.Sprintf("failed to generate unique namespace after %d attempts", maxAttempts), nil)
}
