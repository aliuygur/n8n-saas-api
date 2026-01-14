package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"
)

type CreateInstanceParams struct {
	UserID    string
	Subdomain string
}

func (s *Service) CreateInstance(ctx context.Context, params CreateInstanceParams) (*Instance, error) {
	l := appctx.GetLogger(ctx)

	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_create_%s", params.UserID))
	defer releaseLock()

	// Check if subdomain already exists
	exists, err := queries.CheckSubdomainExists(ctx, params.Subdomain)
	if err != nil {
		return nil, apperrs.Server("failed to check subdomain existence", err)
	}
	if exists {
		return nil, apperrs.Client(apperrs.CodeConflict, "subdomain already taken")
	}

	sub, err := queries.GetSubscriptionByUserID(ctx, params.UserID)
	if err != nil {
		return nil, apperrs.Server("failed to get subscription for user", err)
	}

	if sub.Status == SubscriptionStatusTrial {
		count, err := queries.CountActiveInstancesByUserID(ctx, params.UserID)
		if err != nil {
			return nil, apperrs.Server("failed to count instances", err)
		}
		if count > 0 {
			return nil, apperrs.Client(apperrs.CodeForbidden, "trial users can only have one instance")
		}
	}

	namespace, err := s.generateUniqueNamespace(ctx, queries)
	if err != nil {
		return nil, err
	}

	// Start trial if needed (idempotent, no rollback needed)
	if sub.Status == SubscriptionStatusTrial && !sub.TrialEndsAt.Valid {
		trialEndsAt := time.Now().Add(3 * 24 * time.Hour) // 3 days trial
		_, err = queries.UpdateSubscriptionTrialEndsAt(ctx, db.UpdateSubscriptionTrialEndsAtParams{
			ID: sub.ID,
			TrialEndsAt: pgtype.Timestamp{
				Time:  trialEndsAt,
				Valid: true,
			},
		})
		if err != nil {
			return nil, apperrs.Server("failed to start trial", err)
		}
		l.Debug("started trial subscription", "user_id", params.UserID, "subscription_id", sub.ID, "trial_ends_at", trialEndsAt)
	}

	// Create instance in database
	dbInst, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
		UserID:    params.UserID,
		Namespace: namespace,
		Subdomain: params.Subdomain,
		Status:    InstanceStatusDeployed,
	})
	if err != nil {
		return nil, apperrs.Server("failed to create instance in database", err)
	}
	instanceID := dbInst.ID

	// Increase subscription quantity (skip for trial users)
	quantityIncreased, err := s.increaseSubscriptionQuantity(ctx, queries, params.UserID)
	if err != nil {
		// Rollback: delete instance
		if delErr := queries.DeleteInstance(ctx, instanceID); delErr != nil {
			l.Error("failed to rollback instance creation", "instance_id", instanceID, "error", delErr)
		}
		return nil, err
	}

	// Deploy to GKE
	domain := fmt.Sprintf("https://%s.ranx.cloud", params.Subdomain)
	n8nInstance := &n8ntemplates.N8N_V1{
		Namespace:     namespace,
		EncryptionKey: lo.RandomString(32, lo.AlphanumericCharset),
		BaseURL:       domain,
	}

	if err := s.gke.Apply(ctx, n8nInstance); err != nil {
		// Rollback: decrease quantity if it was increased
		if quantityIncreased {
			s.decreaseSubscriptionQuantity(ctx, queries, params.UserID)
		}
		// Rollback: delete instance
		if delErr := queries.DeleteInstance(ctx, instanceID); delErr != nil {
			l.Error("failed to rollback instance creation", "instance_id", instanceID, "error", delErr)
		}
		return nil, fmt.Errorf("failed to deploy n8n: %w", err)
	}

	// Fetch the full instance from the database to return
	dbInstance, err := queries.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created instance: %w", err)
	}

	instance := toDomainInstance(dbInstance)
	return &instance, nil
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

func (s *Service) increaseSubscriptionQuantity(ctx context.Context, queries *db.Queries, userID string) (bool, error) {
	l := appctx.GetLogger(ctx)

	subscription, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return false, nil
		}
		return false, apperrs.Server("failed to get subscription", err)
	}

	// Skip if trial (no subscription_id) or trial not ended yet
	if subscription.SubscriptionID == "" || (subscription.TrialEndsAt.Valid && time.Now().Before(subscription.TrialEndsAt.Time)) {
		l.Debug("skipping quantity increase for trial user", "user_id", userID)
		return false, nil
	}

	// Fetch subscription from LemonSqueezy to get subscription_item_id
	lsSubscription, err := s.lemonsqueezy.GetSubscription(ctx, subscription.SubscriptionID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch subscription from LemonSqueezy: %w", err)
	}

	if lsSubscription.Data.Attributes.FirstSubscriptionItem == nil {
		l.Debug("subscription has no items, skipping quantity update", "user_id", userID)
		return false, nil
	}

	newQuantity := subscription.Quantity + 1
	subscriptionItemID := lsSubscription.Data.Attributes.FirstSubscriptionItem.ID

	if err := s.lemonsqueezy.UpdateSubscriptionItemQuantity(ctx, subscriptionItemID, newQuantity); err != nil {
		return false, fmt.Errorf("failed to update subscription quantity in LemonSqueezy: %w", err)
	}

	if err := queries.UpdateSubscriptionQuantity(ctx, db.UpdateSubscriptionQuantityParams{
		ID:       subscription.ID,
		Quantity: newQuantity,
	}); err != nil {
		return true, apperrs.Server("failed to update subscription quantity in database", err)
	}

	l.Info("increased subscription quantity", "user_id", userID, "new_quantity", newQuantity)
	return true, nil
}

func (s *Service) decreaseSubscriptionQuantity(ctx context.Context, queries *db.Queries, userID string) {
	l := appctx.GetLogger(ctx)

	subscription, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		l.Error("failed to get subscription for rollback", "user_id", userID, "error", err)
		return
	}

	if subscription.SubscriptionID == "" {
		return
	}

	lsSubscription, err := s.lemonsqueezy.GetSubscription(ctx, subscription.SubscriptionID)
	if err != nil {
		l.Error("failed to fetch subscription from LemonSqueezy for rollback", "error", err)
		return
	}

	if lsSubscription.Data.Attributes.FirstSubscriptionItem == nil {
		return
	}

	newQuantity := subscription.Quantity - 1
	if newQuantity < 1 {
		newQuantity = 1
	}

	subscriptionItemID := lsSubscription.Data.Attributes.FirstSubscriptionItem.ID

	if err := s.lemonsqueezy.UpdateSubscriptionItemQuantity(ctx, subscriptionItemID, newQuantity); err != nil {
		l.Error("failed to revert quantity in LemonSqueezy", "error", err)
		return
	}

	if err := queries.UpdateSubscriptionQuantity(ctx, db.UpdateSubscriptionQuantityParams{
		ID:       subscription.ID,
		Quantity: newQuantity,
	}); err != nil {
		l.Error("failed to revert quantity in database", "error", err)
		return
	}

	l.Debug("reverted subscription quantity", "user_id", userID, "quantity", newQuantity)
}
