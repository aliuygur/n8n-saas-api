package subscription

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/polarsource/polar-go/models/components"
)


// WebhookEvent represents the standard webhook payload structure
type WebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// PolarWebhook handles incoming webhooks from Polar
//
//encore:api public raw method=POST path=/api/webhooks/polar
func (s *Service) PolarWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		rlog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify webhook signature
	if err := s.verifyWebhookSignature(r.Header, bodyBytes); err != nil {
		rlog.Error("Webhook signature verification failed", "error", err)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Parse the webhook event
	var event WebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		rlog.Error("Failed to parse webhook event", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	rlog.Info("Received Polar webhook", "event_type", event.Type)

	// Route to appropriate handler based on event type
	ctx := r.Context()
	var handlerErr error

	switch event.Type {
	case "subscription.created":
		handlerErr = s.handleSubscriptionCreated(ctx, event.Data)
	case "subscription.updated":
		handlerErr = s.handleSubscriptionUpdated(ctx, event.Data)
	case "subscription.active":
		handlerErr = s.handleSubscriptionActive(ctx, event.Data)
	case "subscription.canceled":
		handlerErr = s.handleSubscriptionCanceled(ctx, event.Data)
	case "subscription.revoked":
		handlerErr = s.handleSubscriptionRevoked(ctx, event.Data)
	default:
		rlog.Info("Unhandled webhook event type", "event_type", event.Type)
		w.WriteHeader(http.StatusOK)
		return
	}

	if handlerErr != nil {
		rlog.Error("Failed to handle webhook event", "event_type", event.Type, "error", handlerErr)
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// verifyWebhookSignature verifies the webhook signature using Standard Webhooks spec
// Reference: https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md
func (s *Service) verifyWebhookSignature(headers http.Header, body []byte) error {
	// Extract required headers
	webhookID := headers.Get("webhook-id")
	webhookTimestamp := headers.Get("webhook-timestamp")
	webhookSignature := headers.Get("webhook-signature")

	if webhookID == "" || webhookTimestamp == "" || webhookSignature == "" {
		return fmt.Errorf("missing required webhook headers")
	}

	// Verify timestamp to prevent replay attacks (5 minute tolerance)
	timestamp, err := strconv.ParseInt(webhookTimestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook timestamp: %w", err)
	}

	now := time.Now().Unix()
	if abs(now-timestamp) > 300 { // 5 minutes
		return fmt.Errorf("webhook timestamp too old or too far in future")
	}

	// Construct the signed content as per Standard Webhooks spec
	signedContent := fmt.Sprintf("%s.%s.%s", webhookID, webhookTimestamp, string(body))

	// Base64 decode the webhook secret
	secret, err := base64.StdEncoding.DecodeString(secrets.PolarWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to decode webhook secret: %w", err)
	}

	// Generate expected signature using HMAC-SHA256
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signedContent))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// The signature header contains multiple versions (v1,hash v2,hash etc.)
	// We need to check if any match
	signatures := strings.Split(webhookSignature, " ")
	for _, sig := range signatures {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) == 2 && parts[0] == "v1" {
			if hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
				return nil
			}
		}
	}

	return fmt.Errorf("signature verification failed")
}

// handleSubscriptionCreated handles the subscription.created event
// Note: This event is triggered when a Polar subscription is created via checkout
// We don't automatically create a subscription here because it's already created
// by the checkout callback handler which knows the instance_id
func (s *Service) handleSubscriptionCreated(ctx context.Context, data json.RawMessage) error {
	var subscription components.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	rlog.Info("Processing subscription.created",
		"polar_subscription_id", subscription.ID,
		"customer_id", subscription.CustomerID,
		"status", subscription.Status,
	)

	// The subscription should already be created by the checkout callback
	// This event is just for logging and verification
	queries := db.New(s.db)

	// Try to find existing subscription by polar subscription ID
	// If found, update status; if not found, log warning (checkout callback should have created it)
	status := mapPolarStatusToInternal(subscription.Status)

	if err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: subscription.ID,
		Status:              status,
	}); err != nil {
		rlog.Warn("Could not update subscription from webhook - may not exist yet",
			"polar_subscription_id", subscription.ID,
			"error", err,
		)
		// Don't return error as checkout callback might not have happened yet
		return nil
	}

	rlog.Info("Subscription status updated from webhook",
		"polar_subscription_id", subscription.ID,
		"status", status,
	)

	return nil
}

// handleSubscriptionUpdated handles the subscription.updated event
// This is a catch-all event for various subscription changes
func (s *Service) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var subscription components.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	rlog.Info("Processing subscription.updated",
		"subscription_id", subscription.ID,
		"status", subscription.Status,
	)

	queries := db.New(s.db)
	status := mapPolarStatusToInternal(subscription.Status)

	// Update subscription status by Polar subscription ID
	if err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: subscription.ID,
		Status:              status,
	}); err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	rlog.Info("Subscription updated successfully",
		"subscription_id", subscription.ID,
		"status", status,
	)

	return nil
}

// handleSubscriptionActive handles the subscription.active event
func (s *Service) handleSubscriptionActive(ctx context.Context, data json.RawMessage) error {
	var subscription components.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	rlog.Info("Processing subscription.active",
		"subscription_id", subscription.ID,
	)

	queries := db.New(s.db)

	if err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: subscription.ID,
		Status:              StatusActive,
	}); err != nil {
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	rlog.Info("Subscription activated successfully", "subscription_id", subscription.ID)
	return nil
}

// handleSubscriptionCanceled handles the subscription.canceled event
func (s *Service) handleSubscriptionCanceled(ctx context.Context, data json.RawMessage) error {
	var subscription components.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	rlog.Info("Processing subscription.canceled",
		"subscription_id", subscription.ID,
	)

	queries := db.New(s.db)

	if err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: subscription.ID,
		Status:              StatusCanceled,
	}); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	rlog.Info("Subscription canceled successfully", "subscription_id", subscription.ID)
	return nil
}

// handleSubscriptionRevoked handles the subscription.revoked event
func (s *Service) handleSubscriptionRevoked(ctx context.Context, data json.RawMessage) error {
	var subscription components.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	rlog.Info("Processing subscription.revoked",
		"subscription_id", subscription.ID,
	)

	queries := db.New(s.db)

	// Revoked subscriptions should be marked as expired
	if err := queries.UpdateSubscriptionStatusByPolarID(ctx, db.UpdateSubscriptionStatusByPolarIDParams{
		PolarSubscriptionID: subscription.ID,
		Status:              StatusExpired,
	}); err != nil {
		return fmt.Errorf("failed to revoke subscription: %w", err)
	}

	rlog.Info("Subscription revoked successfully", "subscription_id", subscription.ID)
	return nil
}

// mapPolarStatusToInternal maps Polar subscription status to internal status
func mapPolarStatusToInternal(polarStatus components.SubscriptionStatus) string {
	switch polarStatus {
	case components.SubscriptionStatusTrialing:
		return StatusTrial
	case components.SubscriptionStatusActive:
		return StatusActive
	case components.SubscriptionStatusCanceled:
		return StatusCanceled
	case components.SubscriptionStatusPastDue:
		return StatusPastDue
	case components.SubscriptionStatusUnpaid:
		return StatusPastDue
	case components.SubscriptionStatusIncomplete:
		return StatusNone
	case components.SubscriptionStatusIncompleteExpired:
		return StatusExpired
	default:
		return StatusNone
	}
}

// abs returns absolute value of int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
