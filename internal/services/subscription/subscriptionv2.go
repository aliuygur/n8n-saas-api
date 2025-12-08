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

type GetSubscriptionByUserIDRequest struct {
	UserID string
}

//encore:api private
func (s *Service) GetSubscriptionByUserID(ctx context.Context, req *GetSubscriptionByUserIDRequest) (*Subscription, error) {
	queries := db.New(s.db)

	subscriptionRow, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return &Subscription{
				Status: StatusNone,
			}, nil
		}
		return nil, errs.WrapCode(err, errs.NotFound, "subscription not found")
	}

	subscription := &Subscription{
		ID:                  subscriptionRow.ID,
		UserID:              subscriptionRow.UserID,
		PolarCustomerID:     subscriptionRow.PolarCustomerID,
		PolarProductID:      subscriptionRow.PolarProductID,
		PolarSubscriptionID: subscriptionRow.PolarSubscriptionID,
		Status:              subscriptionRow.Status,
		CreatedAt:           subscriptionRow.CreatedAt,
		UpdatedAt:           subscriptionRow.UpdatedAt,
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
	UserID string
	Days   int // number of days for the trial period (default: 3)
}

//encore:api private
func (s *Service) CreateTrialSubscription(ctx context.Context, req *CreateTrialSubscriptionRequest) (*Subscription, error) {

	if req.Days <= 0 {
		req.Days = DefaultTrialDays
	}

	queries := db.New(s.db)

	trialEndsAt := time.Now().AddDate(0, 0, req.Days)

	// check if subscription already exists
	_, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err == nil {
		return nil, errs.WrapCode(nil, errs.AlreadyExists, "subscription already exists")
	} else if !db.IsNotFoundError(err) {
		return nil, errs.WrapCode(err, errs.Internal, "failed to check existing subscription")
	}

	// create trial subscription

	subscriptionRow, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              req.UserID,
		Status:              StatusTrial,
		PolarCustomerID:     "",
		PolarSubscriptionID: "",
		TrialEndsAt:         sql.NullTime{Time: trialEndsAt, Valid: true},
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create trial subscription")
	}

	subscription := &Subscription{
		ID:                  subscriptionRow.ID,
		UserID:              subscriptionRow.UserID,
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

type IncrementSeatsRequest struct {
	SubscriptionID string
}

//encore:api private
func (s *Service) IncrementSeats(ctx context.Context, req *IncrementSeatsRequest) error {
	queries := db.New(s.db)

	if err := queries.IncrementSubscriptionSeatsByID(ctx, req.SubscriptionID); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to increment seats")
	}

	return nil
}
