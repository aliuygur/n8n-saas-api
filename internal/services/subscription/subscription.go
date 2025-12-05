package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// SubscriptionStatus represents the current subscription state
type SubscriptionStatus struct {
	Status            string     `json:"status"`
	InstanceCount     int        `json:"instance_count"`
	MaxInstances      int        `json:"max_instances"`
	TrialEndsAt       *time.Time `json:"trial_ends_at,omitempty"`
	BillingAnchor     *time.Time `json:"billing_anchor,omitempty"`
	IsActive          bool       `json:"is_active"`
	CanCreateInstance bool       `json:"can_create_instance"`
}

// StartTrialRequest represents the request to start a trial
type StartTrialRequest struct {
	UserID string `json:"user_id"`
}

// StartTrialResponse represents the response after starting a trial
type StartTrialResponse struct {
	SubscriptionID string    `json:"subscription_id"`
	TrialEndsAt    time.Time `json:"trial_ends_at"`
	Status         string    `json:"status"`
}

// StartTrial creates a new trial subscription for a user
// This is called when user creates their first instance
//
//encore:api private
func (s *Service) StartTrial(ctx context.Context, req *StartTrialRequest) (*StartTrialResponse, error) {
	queries := db.New(s.db)

	// Check if user already has a subscription
	existing, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err == nil {
		// User already has a subscription
		return &StartTrialResponse{
			SubscriptionID: existing.ID,
			TrialEndsAt:    existing.TrialEndsAt.Time,
			Status:         existing.Status,
		}, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing subscription: %w", err)
	}

	// Create new trial subscription
	trialStartedAt := time.Now()
	trialEndsAt := trialStartedAt.Add(TrialDurationHours * time.Hour)

	subscription, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:         req.UserID,
		Status:         "trial",
		TrialStartedAt: sql.NullTime{Time: trialStartedAt, Valid: true},
		TrialEndsAt:    sql.NullTime{Time: trialEndsAt, Valid: true},
		InstanceCount:  1, // First instance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trial subscription: %w", err)
	}

	rlog.Info("Trial subscription created",
		"user_id", req.UserID,
		"subscription_id", subscription.ID,
		"trial_ends_at", trialEndsAt,
	)

	return &StartTrialResponse{
		SubscriptionID: subscription.ID,
		TrialEndsAt:    trialEndsAt,
		Status:         subscription.Status,
	}, nil
}

// GetSubscriptionStatusRequest represents the request to get subscription status
type GetSubscriptionStatusRequest struct {
	UserID string `json:"user_id"`
}

// GetSubscriptionStatus returns the current subscription status for a user
//
//encore:api private
func (s *Service) GetSubscriptionStatus(ctx context.Context, req *GetSubscriptionStatusRequest) (*SubscriptionStatus, error) {
	queries := db.New(s.db)

	subscription, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err == sql.ErrNoRows {
		// No subscription yet - user hasn't created any instances
		return &SubscriptionStatus{
			Status:            "none",
			InstanceCount:     0,
			MaxInstances:      1, // Can create first instance to start trial
			IsActive:          true,
			CanCreateInstance: true,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Determine max instances based on status
	maxInstances := 1
	if subscription.Status == "active" {
		// Active subscriptions have unlimited instances (pay-per-instance)
		maxInstances = 999
	}

	// Check if subscription is still valid
	isActive := false
	canCreateInstance := false

	switch subscription.Status {
	case "trial":
		// Check if trial has expired
		if subscription.TrialEndsAt.Valid && time.Now().Before(subscription.TrialEndsAt.Time) {
			isActive = true
			canCreateInstance = int(subscription.InstanceCount) < maxInstances
		} else {
			// Trial expired
			isActive = false
			canCreateInstance = false
		}
	case "active":
		isActive = true
		canCreateInstance = true
	case "expired", "canceled", "past_due":
		isActive = false
		canCreateInstance = false
	}

	status := &SubscriptionStatus{
		Status:            subscription.Status,
		InstanceCount:     int(subscription.InstanceCount),
		MaxInstances:      maxInstances,
		IsActive:          isActive,
		CanCreateInstance: canCreateInstance,
	}

	if subscription.TrialEndsAt.Valid {
		status.TrialEndsAt = &subscription.TrialEndsAt.Time
	}

	if subscription.BillingAnchorDate.Valid {
		status.BillingAnchor = &subscription.BillingAnchorDate.Time
	}

	return status, nil
}

// ValidateInstanceCreationRequest represents the request to validate instance creation
type ValidateInstanceCreationRequest struct {
	UserID string `json:"user_id"`
}

// ValidateInstanceCreationResponse represents the response from validation
type ValidateInstanceCreationResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// ValidateInstanceCreation checks if a user can create a new instance
// Returns error if they cannot create an instance
//
//encore:api private
func (s *Service) ValidateInstanceCreation(ctx context.Context, req *ValidateInstanceCreationRequest) (*ValidateInstanceCreationResponse, error) {
	status, err := s.GetSubscriptionStatus(ctx, &GetSubscriptionStatusRequest{
		UserID: req.UserID,
	})
	if err != nil {
		return nil, err
	}

	// If no subscription, allow first instance (will start trial)
	if status.Status == "none" {
		return &ValidateInstanceCreationResponse{
			Allowed: true,
		}, nil
	}

	// Check if trial has expired
	if status.Status == "trial" && !status.IsActive {
		return nil, &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "Your trial has expired. Please subscribe to continue creating instances.",
		}
	}

	// Check if on trial and trying to create second instance
	if status.Status == "trial" && status.InstanceCount >= 1 {
		return nil, &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "Subscribe to add more instances. Trial users are limited to 1 instance.",
		}
	}

	// Check if subscription is expired/canceled
	if !status.IsActive {
		return nil, &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: fmt.Sprintf("Your subscription is %s. Please update your subscription to continue.", status.Status),
		}
	}

	return &ValidateInstanceCreationResponse{
		Allowed: true,
	}, nil
}

// IncrementInstanceRequest represents the request to increment instance count
type IncrementInstanceRequest struct {
	UserID string `json:"user_id"`
}

// IncrementInstance increments the instance count for a user's subscription
//
//encore:api private
func (s *Service) IncrementInstance(ctx context.Context, req *IncrementInstanceRequest) error {
	queries := db.New(s.db)

	// Increment instance count in database
	if err := queries.IncrementInstanceCount(ctx, req.UserID); err != nil {
		return fmt.Errorf("failed to increment instance count: %w", err)
	}

	// Get updated subscription
	subscription, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	rlog.Info("Instance count incremented",
		"user_id", req.UserID,
		"new_count", subscription.InstanceCount,
		"status", subscription.Status,
	)

	// If active subscription, update Polar subscription quantity
	if subscription.Status == "active" && subscription.PolarSubscriptionID.Valid && subscription.PolarSubscriptionID.String != "" {
		err = s.updatePolarSubscriptionQuantity(ctx, subscription.PolarSubscriptionID.String, int(subscription.InstanceCount))
		if err != nil {
			rlog.Error("Failed to update Polar subscription quantity", "error", err)
			// Don't fail the operation, just log the error
		}
	}

	return nil
}

// DecrementInstanceRequest represents the request to decrement instance count
type DecrementInstanceRequest struct {
	UserID string `json:"user_id"`
}

// DecrementInstance decrements the instance count for a user's subscription
//
//encore:api private
func (s *Service) DecrementInstance(ctx context.Context, req *DecrementInstanceRequest) error {
	queries := db.New(s.db)

	// Decrement instance count in database
	if err := queries.DecrementInstanceCount(ctx, req.UserID); err != nil {
		return fmt.Errorf("failed to decrement instance count: %w", err)
	}

	// Get updated subscription
	subscription, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	rlog.Info("Instance count decremented",
		"user_id", req.UserID,
		"new_count", subscription.InstanceCount,
		"status", subscription.Status,
	)

	// If active subscription, update Polar subscription quantity
	if subscription.Status == "active" && subscription.PolarSubscriptionID.Valid && subscription.PolarSubscriptionID.String != "" {
		err = s.updatePolarSubscriptionQuantity(ctx, subscription.PolarSubscriptionID.String, int(subscription.InstanceCount))
		if err != nil {
			rlog.Error("Failed to update Polar subscription quantity", "error", err)
			// Don't fail the operation, just log the error
		}
	}

	return nil
}

// updatePolarSubscriptionQuantity updates the quantity of a Polar subscription
// This triggers prorated billing adjustments in Polar
func (s *Service) updatePolarSubscriptionQuantity(ctx context.Context, polarSubscriptionID string, newQuantity int) error {
	rlog.Info("Updating Polar subscription quantity",
		"subscription_id", polarSubscriptionID,
		"new_quantity", newQuantity,
	)

	// Note: The Polar Go SDK's subscription update functionality
	// Currently, the SDK may not have a direct method for updating quantities
	// We'll implement this when the SDK supports it or use the HTTP API directly

	// For now, log it as a placeholder
	// TODO: Implement actual Polar API call when SDK supports it or use HTTP client
	rlog.Warn("Polar subscription quantity update not yet implemented",
		"subscription_id", polarSubscriptionID,
		"quantity", newQuantity,
	)

	return nil
}
