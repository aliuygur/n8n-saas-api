package services

import (
	"context"
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

// GetUserSubscription returns the subscription for a user (one subscription per user)
func (s *Service) GetUserSubscription(ctx context.Context, userID string) (*Subscription, error) {
	queries := db.New(s.db)

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
		CreatedAt:      sub.CreatedAt,
		UpdatedAt:      sub.UpdatedAt,
		Quantity:       sub.Quantity,
	}, nil
}
