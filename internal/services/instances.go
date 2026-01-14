package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
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

type DeleteInstanceParams struct {
	UserID     string
	InstanceID string
}

func (s *Service) DeleteInstance(ctx context.Context, params DeleteInstanceParams) error {
	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_delete_%s", params.UserID))
	defer releaseLock()

	l := appctx.GetLogger(ctx)

	instance, err := queries.GetInstance(ctx, params.InstanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.UserID != params.UserID {
		return apperrs.Client(apperrs.CodeForbidden, "user does not own the instance")
	}

	if err := s.gke.DeleteNamespace(ctx, instance.Namespace); err != nil {
		return apperrs.Server("failed to delete namespace from Kubernetes", err)
	}
	l.Debug("deleted namespace from Kubernetes", "namespace", instance.Namespace)

	if err := queries.DeleteInstance(ctx, params.InstanceID); err != nil {
		return apperrs.Server("failed to delete instance from database", err)
	}
	l.Debug("deleted instance from database", "instance_id", params.InstanceID)

	// Decrease subscription quantity if user is not on trial
	if err := s.decreaseSubscriptionQuantityForUser(ctx, queries, params.UserID); err != nil {
		// Log error but don't fail the deletion
		l.Error("failed to decrease subscription quantity", "user_id", params.UserID, "error", err)
	}

	return nil
}

// decreaseSubscriptionQuantityForUser decreases the subscription quantity when an instance is deleted
func (s *Service) decreaseSubscriptionQuantityForUser(ctx context.Context, queries *db.Queries, userID string) error {
	l := appctx.GetLogger(ctx)

	// Get subscription from local database
	subscription, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if db.IsNotFoundError(err) {
			// No subscription, skip quantity update
			return nil
		}
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Skip if trial (no subscription_id) or still in trial period
	if subscription.SubscriptionID == "" || (subscription.TrialEndsAt.Valid && time.Now().Before(subscription.TrialEndsAt.Time)) {
		l.Debug("skipping quantity decrease for trial user", "user_id", userID)
		return nil
	}

	// Don't decrease below 1
	if subscription.Quantity <= 1 {
		l.Debug("subscription quantity already at minimum", "user_id", userID, "quantity", subscription.Quantity)
		return nil
	}

	// Fetch subscription from LemonSqueezy to get subscription_item_id
	lsSubscription, err := s.GetSubscription(ctx, subscription.SubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to fetch subscription from LemonSqueezy: %w", err)
	}

	// Check if subscription has an item
	if lsSubscription.Data.Attributes.FirstSubscriptionItem == nil {
		l.Debug("subscription has no items, skipping quantity update", "user_id", userID)
		return nil
	}

	// Decrease quantity in LemonSqueezy
	newQuantity := subscription.Quantity - 1
	subscriptionItemID := lsSubscription.Data.Attributes.FirstSubscriptionItem.ID

	if err := s.UpdateSubscriptionItemQuantity(ctx, subscriptionItemID, newQuantity); err != nil {
		return fmt.Errorf("failed to update subscription quantity in LemonSqueezy: %w", err)
	}

	// Update quantity in local database
	if err := queries.UpdateSubscriptionQuantity(ctx, db.UpdateSubscriptionQuantityParams{
		ID:       subscription.ID,
		Quantity: newQuantity,
	}); err != nil {
		return fmt.Errorf("failed to update subscription quantity in database: %w", err)
	}

	l.Info("decreased subscription quantity", "user_id", userID, "new_quantity", newQuantity)
	return nil
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
