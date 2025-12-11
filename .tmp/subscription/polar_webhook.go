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
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
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
	case "order.paid":
		handlerErr = s.handleOrderPaid(ctx, event.Data)
	// Add more event types as needed
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

	// Try to base64 decode the webhook secret, if it fails use it as-is
	var secret []byte
	decodedSecret, err := base64.StdEncoding.DecodeString(secrets.POLAR_WEBHOOK_SECRET)
	if err != nil {
		// Secret is not base64 encoded, use it directly
		secret = []byte(secrets.POLAR_WEBHOOK_SECRET)
	} else {
		secret = decodedSecret
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

func (s *Service) handleOrderPaid(ctx context.Context, data json.RawMessage) error {

	queries := db.New(s.db)

	var order components.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return fmt.Errorf("failed to unmarshal order data: %w", err)
	}

	rlog.Info("Processing order.paid",
		"order_id", order.ID,
	)

	if order.BillingReason != components.OrderBillingReasonSubscriptionCreate {
		rlog.Info("Order billing reason is not subscription_create, ignoring",
			"order_id", order.ID,
			"billing_reason", order.BillingReason,
		)
		return nil
	}

	// Check if subscription already exists (idempotency)
	existingSubscription, err := queries.GetSubscriptionByPolarID(ctx, *order.SubscriptionID)
	if err == nil {
		rlog.Info("Subscription already exists, skipping",
			"polar_subscription_id", *order.SubscriptionID,
			"instance_id", existingSubscription.InstanceID,
		)
		return nil
	}

	// Get checkout session from database
	checkoutSession, err := queries.GetCheckoutSessionByPolarID(ctx, *order.CheckoutID)
	if err != nil {
		return fmt.Errorf("failed to fetch checkout session: %w", err)
	}

	// Check if already processed
	if checkoutSession.Status == "completed" {
		rlog.Info("Checkout already processed, skipping",
			"checkout_id", *order.CheckoutID,
			"polar_subscription_id", *order.SubscriptionID,
		)
		return nil
	}

	// Create and deploy the instance
	provisionResp, err := provisioning.CreateInstance(ctx, &provisioning.CreateInstanceRequest{
		InstanceID: checkoutSession.InstanceID,
		UserID:     checkoutSession.UserID,
		Subdomain:  checkoutSession.Subdomain,
		DeployNow:  true,
	})
	if err != nil {
		rlog.Error("Failed to create instance", "error", err, "polar_subscription_id", *order.SubscriptionID)
		// Mark checkout as failed
		_ = queries.UpdateCheckoutSessionStatus(ctx, db.UpdateCheckoutSessionStatusParams{
			ID:     checkoutSession.ID,
			Status: "failed",
		})
		return fmt.Errorf("failed to create instance: %w", err)
	}

	rlog.Info("Instance created and deployed successfully",
		"instance_id", provisionResp.InstanceID,
		"user_id", checkoutSession.UserID,
		"subdomain", checkoutSession.Subdomain,
	)

	// Create active subscription for this instance
	sub, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              checkoutSession.UserID,
		InstanceID:          provisionResp.InstanceID,
		PolarCustomerID:     order.CustomerID,
		PolarSubscriptionID: *order.SubscriptionID,
		PolarProductID:      *order.ProductID,
		Status:              mapPolarStatusToInternal(order.Subscription.Status),
	})
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	err = queries.UpdateCheckoutSessionCompleted(ctx, db.UpdateCheckoutSessionCompletedParams{
		ID:         checkoutSession.ID,
		InstanceID: provisionResp.InstanceID,
	})
	if err != nil {
		rlog.Error("Failed to update checkout session to completed", "error", err)
	}

	rlog.Info("Subscription created successfully from webhook",
		"instance_id", provisionResp.InstanceID,
		"subscription_id", sub.ID,
		"polar_subscription_id", *order.SubscriptionID,
	)

	// Placeholder for handling order.paid event if needed in future
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
