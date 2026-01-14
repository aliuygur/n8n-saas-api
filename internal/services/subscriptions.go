package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Subscription represents a subscription
type Subscription struct {
	ID             string
	UserID         string
	ProductID      string
	CustomerID     string
	SubscriptionID string
	Status         string
	TrialEndsAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Quantity       int32
}

func (s *Subscription) IsTrial() bool {
	return s.Status == SubscriptionStatusTrial || s.Status == SubscriptionStatusTrialing
}

// toDomainSubscription maps a db.Subscription to a Subscription (domain layer)
func toDomainSubscription(sub db.Subscription) *Subscription {
	var trialEndsAt *time.Time
	if sub.TrialEndsAt.Valid {
		trialEndsAt = &sub.TrialEndsAt.Time
	}
	return &Subscription{
		ID:             sub.ID,
		UserID:         sub.UserID,
		ProductID:      sub.ProductID,
		CustomerID:     sub.CustomerID,
		SubscriptionID: sub.SubscriptionID,
		Status:         sub.Status,
		TrialEndsAt:    trialEndsAt,
		CreatedAt:      sub.CreatedAt.Time,
		UpdatedAt:      sub.UpdatedAt.Time,
		Quantity:       sub.Quantity,
	}
}

// GetUserSubscription returns the subscription for a user (one subscription per user)
// SyncSubscriptionQuantity syncs the instance count from our DB to LemonSqueezy subscription quantity.
// It counts active instances (where deleted_at IS NULL) and updates LemonSqueezy if quantities differ.
func (s *Service) SyncSubscriptionQuantity(ctx context.Context, userID string) error {
	l := appctx.GetLogger(ctx)
	queries := s.getDB()

	// Get subscription
	sub, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Skip if no LemonSqueezy subscription ID (e.g., trial users)
	if sub.SubscriptionID == "" {
		l.Debug("subscription has no LemonSqueezy subscription ID, skipping sync", "user_id", userID)
		return nil
	}

	// Count active instances in our DB
	instanceCount, err := queries.CountActiveInstancesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to count instances: %w", err)
	}

	l.Debug("syncing subscription quantity", "user_id", userID, "db_instance_count", instanceCount)

	// Fetch subscription from LemonSqueezy
	lsSub, err := s.lemonsqueezy.GetSubscription(ctx, sub.SubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get LemonSqueezy subscription: %w", err)
	}

	// Get LemonSqueezy quantity
	lsQuantity := 0
	if lsSub.Data.Attributes.FirstSubscriptionItem != nil {
		lsQuantity = lsSub.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	l.Debug("fetched LemonSqueezy subscription", "user_id", userID, "ls_quantity", lsQuantity)

	// If our DB count is zero, we might want to handle cancellation or alerts here.
	// For now, we just log it.
	if instanceCount == 0 && lsQuantity == 1 {
		l.Info("user has zero active instances, consider handling cancellation", "user_id", userID)
	}

	// If quantities match, nothing to do
	if int64(lsQuantity) == instanceCount {
		l.Debug("subscription quantities match, no update needed", "user_id", userID)
		return nil
	}

	// Update LemonSqueezy with our DB count (trust our DB)
	if err := s.lemonsqueezy.UpdateSubscriptionQuantity(ctx, sub.SubscriptionID, int32(instanceCount)); err != nil {
		return fmt.Errorf("failed to update LemonSqueezy quantity: %w", err)
	}

	// Update our DB subscription quantity
	if err := queries.UpdateSubscriptionQuantity(ctx, db.UpdateSubscriptionQuantityParams{
		ID:       sub.ID,
		Quantity: int32(instanceCount),
	}); err != nil {
		return fmt.Errorf("failed to update subscription quantity in DB: %w", err)
	}

	l.Debug("updated LemonSqueezy subscription quantity", "user_id", userID, "new_quantity", instanceCount)

	return nil
}

func (s *Service) GetUserSubscription(ctx context.Context, userID string) (*Subscription, error) {
	queries := s.getDB()

	sub, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return toDomainSubscription(sub), nil
}
