package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

// Subscription status constants
const (
	StatusNone     = "none"
	StatusTrial    = "trial"
	StatusActive   = "active"
	StatusExpired  = "expired"
	StatusCanceled = "canceled"
	StatusPastDue  = "past_due"
)

// CreateCheckoutRequest represents the request to create a checkout session
type CreateCheckoutRequest struct {
	UserID     string
	InstanceID string
	Subdomain  string
	UserEmail  string
	SuccessURL string
	ReturnURL  string
}

// CreateCheckoutResponse represents the response with checkout URL
type CreateCheckoutResponse struct {
	CheckoutURL string
	CheckoutID  string
}

// CheckoutSession represents a checkout session
type CheckoutSession struct {
	CheckoutID string
	UserID     string
	InstanceID string
	Status     string
	Subdomain  string
}

// WebhookEvent represents the standard webhook payload structure
type WebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// createCheckoutInternal creates a Polar checkout session
func (h *Handler) createCheckoutInternal(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	checkoutCreate := components.CheckoutCreate{
		Products:           []string{h.config.Polar.ProductID},
		ExternalCustomerID: polargo.Pointer(req.UserID),
		CustomerEmail:      polargo.Pointer(req.UserEmail),
		SuccessURL:         polargo.Pointer(req.SuccessURL),
	}

	if req.ReturnURL != "" {
		checkoutCreate.ReturnURL = polargo.Pointer(req.ReturnURL)
	}

	h.logger.Info("Creating Polar checkout session",
		slog.String("user_id", req.UserID),
		slog.String("subdomain", req.Subdomain),
		slog.String("email", req.UserEmail))

	resp, err := h.polarClient.Checkouts.Create(ctx, checkoutCreate)
	if err != nil {
		h.logger.Error("Failed to create Polar checkout", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	if resp.Checkout == nil {
		return nil, fmt.Errorf("checkout response is nil")
	}

	h.logger.Info("Polar checkout created successfully",
		slog.String("checkout_id", resp.Checkout.ID),
		slog.String("checkout_url", resp.Checkout.URL))

	// Store checkout session in database
	checkoutSession, err := h.db.CreateCheckoutSession(ctx, db.CreateCheckoutSessionParams{
		UserID:          req.UserID,
		InstanceID:      req.InstanceID,
		PolarCheckoutID: resp.Checkout.ID,
		Subdomain:       req.Subdomain,
		UserEmail:       req.UserEmail,
		SuccessUrl:      req.SuccessURL,
		ReturnUrl:       req.ReturnURL,
		Status:          "pending",
	})
	if err != nil {
		h.logger.Error("Failed to store checkout session in database", slog.Any("error", err))
		return nil, fmt.Errorf("failed to store checkout session: %w", err)
	}

	h.logger.Info("Checkout session stored in database",
		slog.String("checkout_session_id", checkoutSession.ID),
		slog.String("polar_checkout_id", checkoutSession.PolarCheckoutID))

	return &CreateCheckoutResponse{
		CheckoutURL: resp.Checkout.URL,
		CheckoutID:  resp.Checkout.ID,
	}, nil
}

// getCheckoutSessionByPolarIDInternal retrieves a checkout session by Polar ID
func (h *Handler) getCheckoutSessionByPolarIDInternal(ctx context.Context, polarCheckoutID string) (*CheckoutSession, error) {
	checkoutSession, err := h.db.GetCheckoutSessionByPolarID(ctx, polarCheckoutID)
	if err != nil {
		h.logger.Error("Failed to get checkout session from database",
			slog.Any("error", err),
			slog.String("polar_checkout_id", polarCheckoutID))
		return nil, fmt.Errorf("failed to get checkout session: %w", err)
	}

	return &CheckoutSession{
		CheckoutID: checkoutSession.PolarCheckoutID,
		UserID:     checkoutSession.UserID,
		InstanceID: checkoutSession.InstanceID,
		Status:     checkoutSession.Status,
		Subdomain:  checkoutSession.Subdomain,
	}, nil
}

// deleteSubscriptionByInstanceIDInternal deletes a subscription by instance ID
func (h *Handler) deleteSubscriptionByInstanceIDInternal(ctx context.Context, instanceID string) error {
	return h.db.DeleteSubscriptionByInstanceID(ctx, instanceID)
}

// PolarWebhook handles incoming webhooks from Polar
func (h *Handler) PolarWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", slog.Any("error", err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify webhook signature
	if err := h.verifyWebhookSignature(r.Header, bodyBytes); err != nil {
		h.logger.Error("Webhook signature verification failed", slog.Any("error", err))
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Parse the webhook event
	var event WebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		h.logger.Error("Failed to parse webhook event", slog.Any("error", err))
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	h.logger.Info("Received Polar webhook", slog.String("event_type", event.Type))

	// Route to appropriate handler based on event type
	ctx := r.Context()
	var handlerErr error

	switch event.Type {
	case "order.paid":
		handlerErr = h.handleOrderPaid(ctx, event.Data)
	default:
		h.logger.Info("Unhandled webhook event type", slog.String("event_type", event.Type))
		w.WriteHeader(http.StatusOK)
		return
	}

	if handlerErr != nil {
		h.logger.Error("Failed to handle webhook event",
			slog.String("event_type", event.Type),
			slog.Any("error", handlerErr))
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// verifyWebhookSignature verifies the webhook signature using Standard Webhooks spec
func (h *Handler) verifyWebhookSignature(headers http.Header, body []byte) error {
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
	decodedSecret, err := base64.StdEncoding.DecodeString(h.polarWebhookSecret)
	if err != nil {
		// Secret is not base64 encoded, use it directly
		secret = []byte(h.polarWebhookSecret)
	} else {
		secret = decodedSecret
	}

	// Generate expected signature using HMAC-SHA256
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signedContent))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// The signature header contains multiple versions (v1,hash v2,hash etc.)
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

// handleOrderPaid handles the order.paid webhook event
func (h *Handler) handleOrderPaid(ctx context.Context, data json.RawMessage) error {
	var order components.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return fmt.Errorf("failed to unmarshal order data: %w", err)
	}

	h.logger.Info("Processing order.paid", slog.String("order_id", order.ID))

	if order.BillingReason != components.OrderBillingReasonSubscriptionCreate {
		h.logger.Info("Order billing reason is not subscription_create, ignoring",
			slog.String("order_id", order.ID),
			slog.String("billing_reason", string(order.BillingReason)))
		return nil
	}

	// Check if subscription already exists (idempotency)
	existingSubscription, err := h.db.GetSubscriptionByPolarID(ctx, *order.SubscriptionID)
	if err == nil {
		h.logger.Info("Subscription already exists, skipping",
			slog.String("polar_subscription_id", *order.SubscriptionID),
			slog.String("instance_id", existingSubscription.InstanceID))
		return nil
	}

	// Get checkout session from database
	checkoutSession, err := h.db.GetCheckoutSessionByPolarID(ctx, *order.CheckoutID)
	if err != nil {
		return fmt.Errorf("failed to fetch checkout session: %w", err)
	}

	// Check if already processed
	if checkoutSession.Status == "completed" {
		h.logger.Info("Checkout already processed, skipping",
			slog.String("checkout_id", *order.CheckoutID),
			slog.String("polar_subscription_id", *order.SubscriptionID))
		return nil
	}

	// Create and deploy the instance (DeployNow removed)
	instance, err := h.createInstanceInternal(ctx, CreateInstanceRequest{
		InstanceID: checkoutSession.InstanceID,
		UserID:     checkoutSession.UserID,
		Subdomain:  checkoutSession.Subdomain,
	})
	if err != nil {
		h.logger.Error("Failed to create instance",
			slog.Any("error", err),
			slog.String("polar_subscription_id", *order.SubscriptionID))
		// Mark checkout as failed
		_ = h.db.UpdateCheckoutSessionStatus(ctx, db.UpdateCheckoutSessionStatusParams{
			ID:     checkoutSession.ID,
			Status: "failed",
		})
		return fmt.Errorf("failed to create instance: %w", err)
	}

	h.logger.Info("Instance created and deployed successfully",
		slog.String("instance_id", instance.ID),
		slog.String("user_id", checkoutSession.UserID),
		slog.String("subdomain", checkoutSession.Subdomain))

	// Create active subscription for this instance
	sub, err := h.db.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              checkoutSession.UserID,
		InstanceID:          instance.ID,
		PolarCustomerID:     order.CustomerID,
		PolarSubscriptionID: *order.SubscriptionID,
		PolarProductID:      *order.ProductID,
		Status:              mapPolarStatusToInternal(order.Subscription.Status),
	})
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	err = h.db.UpdateCheckoutSessionCompleted(ctx, db.UpdateCheckoutSessionCompletedParams{
		ID:         checkoutSession.ID,
		InstanceID: instance.ID,
	})
	if err != nil {
		h.logger.Error("Failed to update checkout session to completed", slog.Any("error", err))
	}

	h.logger.Info("Subscription created successfully from webhook",
		slog.String("instance_id", instance.ID),
		slog.String("subscription_id", sub.ID),
		slog.String("polar_subscription_id", *order.SubscriptionID))

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
