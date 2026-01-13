package services

import (
	"context"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Subscription represents a subscription
type Subscription struct {
	ID                  string
	UserID              string
	InstanceID          string
	ProductID      string
	CustomerID     string
	SubscriptionID string
	Status              string
	TrialEndsAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// GetUserSubscriptions returns all subscriptions for a user
func (s *Service) GetUserSubscriptions(ctx context.Context, userID string) ([]Subscription, error) {
	queries := db.New(s.db)

	dbSubscriptions, err := queries.GetAllSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]Subscription, len(dbSubscriptions))
	for i, sub := range dbSubscriptions {
		var trialEndsAt *time.Time
		if sub.TrialEndsAt.Valid {
			trialEndsAt = &sub.TrialEndsAt.Time
		}

		subscriptions[i] = Subscription{
			ID:                  sub.ID,
			UserID:              sub.UserID,
			InstanceID:          sub.InstanceID,
			ProductID:      sub.ProductID,
			CustomerID:     sub.CustomerID,
			SubscriptionID: sub.SubscriptionID,
			Status:              sub.Status,
			TrialEndsAt:         trialEndsAt,
			CreatedAt:           sub.CreatedAt,
			UpdatedAt:           sub.UpdatedAt,
		}
	}

	return subscriptions, nil
}
