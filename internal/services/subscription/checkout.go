package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

// CreateCheckoutRequest represents the request to create a checkout session
type CreateCheckoutRequest struct {
	UserID     string `json:"user_id"`
	SuccessURL string `json:"success_url"`
	ReturnURL  string `json:"return_url,omitempty"`
}

// CreateCheckoutResponse represents the response with checkout URL
type CreateCheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
	CheckoutID  string `json:"checkout_id"`
}

// CreateCheckout creates a Polar checkout session for subscription activation
//
//encore:api private
func (s *Service) CreateCheckout(ctx context.Context, req *CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	queries := db.New(s.db)

	// Get subscription
	subscription, err := queries.GetSubscriptionByUserID(ctx, req.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no subscription found for user")
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Only allow checkout for trial or expired subscriptions
	if subscription.Status != "trial" && subscription.Status != "expired" {
		return nil, fmt.Errorf("subscription is already active")
	}

	// Get user email for Polar customer
	user, err := queries.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create checkout session with Polar
	// The quantity is the current instance count (at least 1 for trial users)
	quantity := max(subscription.InstanceCount, 1)

	checkoutCreate := components.CheckoutCreate{
		Products:      []string{secrets.PolarProductID},
		CustomerEmail: polargo.Pointer(user.Email),
		CustomerName:  polargo.Pointer(user.Name),
		SuccessURL:    polargo.Pointer(req.SuccessURL),
	}

	if req.ReturnURL != "" {
		checkoutCreate.ReturnURL = polargo.Pointer(req.ReturnURL)
	}

	rlog.Info("Creating Polar checkout session",
		"user_id", req.UserID,
		"email", user.Email,
		"quantity", quantity,
	)

	resp, err := s.polarClient.Checkouts.Create(ctx, checkoutCreate)
	if err != nil {
		rlog.Error("Failed to create Polar checkout", "error", err)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	if resp.Checkout == nil {
		return nil, fmt.Errorf("checkout response is nil")
	}

	rlog.Info("Polar checkout created successfully",
		"checkout_id", resp.Checkout.ID,
		"checkout_url", resp.Checkout.URL,
	)

	return &CreateCheckoutResponse{
		CheckoutURL: resp.Checkout.URL,
		CheckoutID:  resp.Checkout.ID,
	}, nil
}

// WebhookRequest represents incoming webhook from Polar
// Using json.RawMessage to handle dynamic webhook payloads
type WebhookRequest struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// WebhookResponse represents the webhook acknowledgment
type WebhookResponse struct {
	Success bool `json:"success"`
}

// HandleWebhook processes Polar webhook events
//
//encore:api public method=POST path=/subscription/webhook
func (s *Service) HandleWebhook(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error) {
	rlog.Info("Received Polar webhook", "event", req.Event)

	// Parse the webhook data
	var data map[string]interface{}
	if err := json.Unmarshal(req.Data, &data); err != nil {
		rlog.Error("Failed to parse webhook data", "error", err)
		return &WebhookResponse{Success: false}, err
	}

	switch req.Event {
	case "checkout.session.completed", "checkout.completed":
		return s.handleCheckoutCompleted(ctx, data)
	case "subscription.created":
		return s.handleSubscriptionCreated(ctx, data)
	case "subscription.updated":
		return s.handleSubscriptionUpdated(ctx, data)
	case "subscription.canceled", "subscription.cancelled":
		return s.handleSubscriptionCanceled(ctx, data)
	default:
		rlog.Info("Unhandled webhook event", "event", req.Event)
		return &WebhookResponse{Success: true}, nil
	}
}

// handleCheckoutCompleted processes successful checkout completion
func (s *Service) handleCheckoutCompleted(ctx context.Context, data map[string]interface{}) (*WebhookResponse, error) {
	// Extract metadata
	metadata, ok := data["metadata"].(map[string]interface{})
	if !ok {
		rlog.Warn("No metadata in checkout completion webhook")
		return &WebhookResponse{Success: false}, fmt.Errorf("missing metadata")
	}

	userID, ok := metadata["user_id"].(string)
	if !ok {
		rlog.Warn("No user_id in metadata")
		return &WebhookResponse{Success: false}, fmt.Errorf("missing user_id")
	}

	// Extract customer and subscription IDs
	polarCustomerID := ""
	if customerID, ok := data["customer_id"].(string); ok {
		polarCustomerID = customerID
	}

	polarSubscriptionID := ""
	if subscriptionID, ok := data["subscription_id"].(string); ok {
		polarSubscriptionID = subscriptionID
	}

	rlog.Info("Processing checkout completion",
		"user_id", userID,
		"polar_customer_id", polarCustomerID,
		"polar_subscription_id", polarSubscriptionID,
	)

	// Update subscription to active
	queries := db.New(s.db)

	// Update status
	err := queries.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
		UserID: userID,
		Status: "active",
	})
	if err != nil {
		rlog.Error("Failed to update subscription status", "error", err)
		return &WebhookResponse{Success: false}, err
	}

	// Update Polar info
	billingAnchor := time.Now()
	err = queries.UpdateSubscriptionPolarInfo(ctx, db.UpdateSubscriptionPolarInfoParams{
		UserID:              userID,
		PolarCustomerID:     sql.NullString{String: polarCustomerID, Valid: polarCustomerID != ""},
		PolarSubscriptionID: sql.NullString{String: polarSubscriptionID, Valid: polarSubscriptionID != ""},
		BillingAnchorDate:   sql.NullTime{Time: billingAnchor, Valid: true},
	})
	if err != nil {
		rlog.Error("Failed to update Polar info", "error", err)
		return &WebhookResponse{Success: false}, err
	}

	rlog.Info("Subscription activated successfully", "user_id", userID)

	return &WebhookResponse{Success: true}, nil
}

// handleSubscriptionCreated processes new subscription creation
func (s *Service) handleSubscriptionCreated(ctx context.Context, data map[string]interface{}) (*WebhookResponse, error) {
	rlog.Info("Subscription created event received")
	// This is usually handled by checkout completion
	return &WebhookResponse{Success: true}, nil
}

// handleSubscriptionUpdated processes subscription updates
func (s *Service) handleSubscriptionUpdated(ctx context.Context, data map[string]any) (*WebhookResponse, error) {
	rlog.Info("Subscription updated event received")

	// Extract subscription ID and status
	subscriptionID, ok := data["id"].(string)
	if !ok {
		return &WebhookResponse{Success: false}, fmt.Errorf("missing subscription id")
	}

	status, ok := data["status"].(string)
	if !ok {
		rlog.Warn("No status in subscription update")
		return &WebhookResponse{Success: true}, nil
	}

	rlog.Info("Subscription status updated",
		"subscription_id", subscriptionID,
		"status", status,
	)

	// Map Polar status to our status
	// Polar statuses: active, canceled, incomplete, incomplete_expired, past_due, trialing, unpaid
	var ourStatus string
	switch status {
	case "active", "trialing":
		ourStatus = "active"
	case "past_due", "unpaid":
		ourStatus = "past_due"
	case "canceled", "incomplete_expired":
		ourStatus = "canceled"
	default:
		ourStatus = status
	}

	// Find subscription by Polar ID and update status
	queries := db.New(s.db)
	err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: sql.NullString{String: subscriptionID, Valid: true},
		Status:              ourStatus,
	})
	if err != nil {
		rlog.Error("Failed to update subscription status", "error", err)
		return &WebhookResponse{Success: false}, err
	}

	rlog.Info("Subscription status updated successfully",
		"polar_subscription_id", subscriptionID,
		"new_status", ourStatus,
	)

	return &WebhookResponse{Success: true}, nil
}

// handleSubscriptionCanceled processes subscription cancellation
func (s *Service) handleSubscriptionCanceled(ctx context.Context, data map[string]interface{}) (*WebhookResponse, error) {
	subscriptionID, ok := data["id"].(string)
	if !ok {
		return &WebhookResponse{Success: false}, fmt.Errorf("missing subscription id")
	}

	rlog.Info("Subscription canceled",
		"subscription_id", subscriptionID,
	)

	// TODO: Update subscription status to canceled
	// Need query to find by polar_subscription_id

	return &WebhookResponse{Success: true}, nil
}
