package services

import (
	"context"
	"fmt"
	"time"

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

// GetUserSubscription returns the subscription for a user (one subscription per user)
// SyncSubscriptionQuantity syncs the instance count from our DB to LemonSqueezy subscription quantity.
// It counts active instances (where deleted_at IS NULL) and updates LemonSqueezy if quantities differ.
func (s *Service) SyncSubscriptionQuantity(ctx context.Context, userID string) error {
	queries := s.getDB()

	// Get subscription
	sub, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Skip if no LemonSqueezy subscription ID (e.g., trial users)
	if sub.SubscriptionID == "" {
		return nil
	}

	// Count active instances in our DB
	instanceCount, err := queries.CountActiveInstancesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to count instances: %w", err)
	}

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

	// If quantities match, nothing to do
	if int64(lsQuantity) == instanceCount {
		return nil
	}

	// Update LemonSqueezy with our DB count (trust our DB)
	if err := s.lemonsqueezy.UpdateSubscriptionQuantity(ctx, sub.SubscriptionID, int32(instanceCount)); err != nil {
		return fmt.Errorf("failed to update LemonSqueezy quantity: %w", err)
	}

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
	}, nil
}
