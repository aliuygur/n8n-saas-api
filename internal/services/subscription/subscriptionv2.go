package subscription

import (
	"context"
	"database/sql"
	"time"

	"encore.dev/beta/errs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

const (
	StatusNone     = "none"
	StatusTrial    = "trial"
	StatusActive   = "active"
	StatusPastDue  = "past_due"
	StatusCanceled = "canceled"
	StatusExpired  = "expired"

	DefaultTrialDays = 3
)

type Subscription struct {
	ID                  string
	UserID              string
	InstanceID          string
	Status              string
	PolarSubscriptionID string
	PolarCustomerID     string
	PolarProductID      string
	TrialEndsAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive
}

func (s *Subscription) IsTrial() bool {
	return s.Status == StatusTrial
}

func (s *Subscription) CanCreateInstance() bool {
	return s.Status == StatusActive || s.Status == StatusNone
}

type GetAllSubscriptionsByUserIDRequest struct {
	UserID string
}

type GetAllSubscriptionsByUserIDResponse struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

//encore:api private
func (s *Service) GetAllSubscriptionsByUserID(ctx context.Context, req *GetAllSubscriptionsByUserIDRequest) (*GetAllSubscriptionsByUserIDResponse, error) {
	queries := db.New(s.db)

	subscriptionRows, err := queries.GetAllSubscriptionsByUserID(ctx, req.UserID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get subscriptions")
	}

	if len(subscriptionRows) == 0 {
		return &GetAllSubscriptionsByUserIDResponse{
			Subscriptions: []Subscription{},
		}, nil
	}

	subscriptions := make([]Subscription, len(subscriptionRows))
	for i, row := range subscriptionRows {
		subscriptions[i] = Subscription{
			ID:                  row.ID,
			UserID:              row.UserID,
			InstanceID:          row.InstanceID,
			PolarCustomerID:     row.PolarCustomerID,
			PolarProductID:      row.PolarProductID,
			PolarSubscriptionID: row.PolarSubscriptionID,
			Status:              row.Status,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
		}
		if row.TrialEndsAt.Valid {
			subscriptions[i].TrialEndsAt = &row.TrialEndsAt.Time
		}
	}

	return &GetAllSubscriptionsByUserIDResponse{
		Subscriptions: subscriptions,
	}, nil
}

type GetSubscriptionByInstanceIDRequest struct {
	InstanceID string
}

//encore:api private
func (s *Service) GetSubscriptionByInstanceID(ctx context.Context, req *GetSubscriptionByInstanceIDRequest) (*Subscription, error) {
	queries := db.New(s.db)

	subscriptionRow, err := queries.GetSubscriptionByInstanceID(ctx, req.InstanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, errs.WrapCode(err, errs.NotFound, "subscription not found")
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to get subscription")
	}

	subscription := &Subscription{
		ID:                  subscriptionRow.ID,
		UserID:              subscriptionRow.UserID,
		InstanceID:          subscriptionRow.InstanceID,
		PolarCustomerID:     subscriptionRow.PolarCustomerID,
		PolarProductID:      subscriptionRow.PolarProductID,
		PolarSubscriptionID: subscriptionRow.PolarSubscriptionID,
		Status:              subscriptionRow.Status,
		CreatedAt:           subscriptionRow.CreatedAt,
		UpdatedAt:           subscriptionRow.UpdatedAt,
	}

	if subscriptionRow.TrialEndsAt.Valid {
		subscription.TrialEndsAt = &subscriptionRow.TrialEndsAt.Time
	}

	return subscription, nil
}

type CreateSubscriptionParams struct {
	UserID              string
	Status              string
	PolarSubscriptionID string
	PolarCustomerID     string
}

type CreateTrialSubscriptionRequest struct {
	UserID     string
	InstanceID string
	Days       int // number of days for the trial period (default: 3)
}

//encore:api private
func (s *Service) CreateTrialSubscription(ctx context.Context, req *CreateTrialSubscriptionRequest) (*Subscription, error) {

	if req.Days <= 0 {
		req.Days = DefaultTrialDays
	}

	queries := db.New(s.db)

	trialEndsAt := time.Now().AddDate(0, 0, req.Days)

	// create trial subscription
	subscriptionRow, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              req.UserID,
		InstanceID:          req.InstanceID,
		Status:              StatusTrial,
		PolarCustomerID:     "",
		PolarSubscriptionID: "",
		PolarProductID:      "",
		TrialEndsAt:         sql.NullTime{Time: trialEndsAt, Valid: true},
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create trial subscription")
	}

	subscription := &Subscription{
		ID:                  subscriptionRow.ID,
		UserID:              subscriptionRow.UserID,
		InstanceID:          subscriptionRow.InstanceID,
		PolarCustomerID:     subscriptionRow.PolarCustomerID,
		PolarProductID:      subscriptionRow.PolarProductID,
		PolarSubscriptionID: subscriptionRow.PolarSubscriptionID,
		Status:              subscriptionRow.Status,
		CreatedAt:           subscriptionRow.CreatedAt,
		UpdatedAt:           subscriptionRow.UpdatedAt,
		TrialEndsAt:         &trialEndsAt,
	}

	return subscription, nil
}

type CreateActiveSubscriptionRequest struct {
	UserID              string
	InstanceID          string
	PolarCustomerID     string
	PolarSubscriptionID string
	PolarProductID      string
}

//encore:api private
func (s *Service) CreateActiveSubscription(ctx context.Context, req *CreateActiveSubscriptionRequest) (*Subscription, error) {
	queries := db.New(s.db)

	subscriptionRow, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              req.UserID,
		InstanceID:          req.InstanceID,
		Status:              StatusActive,
		PolarCustomerID:     req.PolarCustomerID,
		PolarSubscriptionID: req.PolarSubscriptionID,
		PolarProductID:      req.PolarProductID,
		TrialEndsAt:         sql.NullTime{Valid: false},
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create subscription")
	}

	subscription := &Subscription{
		ID:                  subscriptionRow.ID,
		UserID:              subscriptionRow.UserID,
		InstanceID:          subscriptionRow.InstanceID,
		PolarCustomerID:     subscriptionRow.PolarCustomerID,
		PolarProductID:      subscriptionRow.PolarProductID,
		PolarSubscriptionID: subscriptionRow.PolarSubscriptionID,
		Status:              subscriptionRow.Status,
		CreatedAt:           subscriptionRow.CreatedAt,
		UpdatedAt:           subscriptionRow.UpdatedAt,
	}

	return subscription, nil
}

type DeleteSubscriptionByInstanceIDRequest struct {
	InstanceID string
}

//encore:api private
func (s *Service) DeleteSubscriptionByInstanceID(ctx context.Context, req *DeleteSubscriptionByInstanceIDRequest) error {
	queries := db.New(s.db)

	if err := queries.DeleteSubscriptionByInstanceID(ctx, req.InstanceID); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete subscription")
	}

	return nil
}
