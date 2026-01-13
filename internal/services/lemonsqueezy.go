package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// LemonSqueezy webhook event types
const (
	LemonSqueezyEventOrderCreated                 = "order_created"
	LemonSqueezyEventSubscriptionCreated          = "subscription_created"
	LemonSqueezyEventSubscriptionUpdated          = "subscription_updated"
	LemonSqueezyEventSubscriptionCancelled        = "subscription_cancelled"
	LemonSqueezyEventSubscriptionResumed          = "subscription_resumed"
	LemonSqueezyEventSubscriptionExpired          = "subscription_expired"
	LemonSqueezyEventSubscriptionPaused           = "subscription_paused"
	LemonSqueezyEventSubscriptionUnpaused         = "subscription_unpaused"
	LemonSqueezyEventSubscriptionPaymentSuccess   = "subscription_payment_success"
	LemonSqueezyEventSubscriptionPaymentFailed    = "subscription_payment_failed"
	LemonSqueezyEventSubscriptionPaymentRecovered = "subscription_payment_recovered"
)

// LemonSqueezy subscription statuses
const (
	LemonSqueezyStatusOnTrial   = "on_trial"
	LemonSqueezyStatusActive    = "active"
	LemonSqueezyStatusPaused    = "paused"
	LemonSqueezyStatusPastDue   = "past_due"
	LemonSqueezyStatusUnpaid    = "unpaid"
	LemonSqueezyStatusCancelled = "cancelled"
	LemonSqueezyStatusExpired   = "expired"
)

// LemonSqueezyWebhookPayload represents the webhook payload from Lemon Squeezy
type LemonSqueezyWebhookPayload struct {
	Meta struct {
		EventName  string     `json:"event_name"`
		WebhookID  string     `json:"webhook_id"`
		TestMode   bool       `json:"test_mode"`
		CustomData CustomData `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			StoreID                int                    `json:"store_id"`
			CustomerID             int                    `json:"customer_id"`
			OrderID                int                    `json:"order_id"`
			OrderItemID            int                    `json:"order_item_id"`
			ProductID              int                    `json:"product_id"`
			VariantID              int                    `json:"variant_id"`
			ProductName            string                 `json:"product_name"`
			VariantName            string                 `json:"variant_name"`
			UserName               string                 `json:"user_name"`
			UserEmail              string                 `json:"user_email"`
			Status                 string                 `json:"status"`
			StatusFormatted        string                 `json:"status_formatted"`
			CardBrand              string                 `json:"card_brand"`
			CardLastFour           string                 `json:"card_last_four"`
			Pause                  *PauseInfo             `json:"pause"`
			Cancelled              bool                   `json:"cancelled"`
			TrialEndsAt            *string                `json:"trial_ends_at"`
			BillingAnchor          int                    `json:"billing_anchor"`
			FirstSubscriptionItem  *SubscriptionItem      `json:"first_subscription_item"`
			Urls                   struct {
				UpdatePaymentMethod string `json:"update_payment_method"`
				CustomerPortal      string `json:"customer_portal"`
			} `json:"urls"`
			RenewsAt  *string `json:"renews_at"`
			EndsAt    *string `json:"ends_at"`
			CreatedAt string  `json:"created_at"`
			UpdatedAt string  `json:"updated_at"`
			TestMode  bool    `json:"test_mode"`
		} `json:"attributes"`
		Relationships struct {
			Store struct {
				Links struct {
					Related string `json:"related"`
					Self    string `json:"self"`
				} `json:"links"`
			} `json:"store"`
			Customer struct {
				Links struct {
					Related string `json:"related"`
					Self    string `json:"self"`
				} `json:"links"`
			} `json:"customer"`
		} `json:"relationships"`
		Links struct {
			Self string `json:"self"`
		} `json:"links"`
	} `json:"data"`
}

// CustomData represents custom data passed in the webhook
type CustomData struct {
	UserID     string `json:"user_id"`
	InstanceID string `json:"instance_id"`
}

// PauseInfo represents pause information for a subscription
type PauseInfo struct {
	Mode      string `json:"mode"`
	ResumesAt string `json:"resumes_at"`
}

// SubscriptionItem represents a subscription item
type SubscriptionItem struct {
	ID        int    `json:"id"`
	ProductID int    `json:"product_id"`
	VariantID int    `json:"variant_id"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// VerifyLemonSqueezySignature verifies the webhook signature
func (s *Service) VerifyLemonSqueezySignature(payload []byte, signature string, webhookSecret string) bool {
	if webhookSecret == "" {
		// In development, you might want to skip verification
		return true
	}

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// HandleLemonSqueezyEvent handles different webhook events
func (s *Service) HandleLemonSqueezyEvent(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)

	switch payload.Meta.EventName {
	case LemonSqueezyEventSubscriptionCreated:
		return s.handleSubscriptionCreated(ctx, payload)

	case LemonSqueezyEventSubscriptionUpdated:
		return s.handleSubscriptionUpdated(ctx, payload)

	case LemonSqueezyEventSubscriptionCancelled:
		return s.handleSubscriptionCancelled(ctx, payload)

	case LemonSqueezyEventSubscriptionResumed:
		return s.handleSubscriptionResumed(ctx, payload)

	case LemonSqueezyEventSubscriptionExpired:
		return s.handleSubscriptionExpired(ctx, payload)

	case LemonSqueezyEventSubscriptionPaused:
		return s.handleSubscriptionPaused(ctx, payload)

	case LemonSqueezyEventSubscriptionUnpaused:
		return s.handleSubscriptionUnpaused(ctx, payload)

	case LemonSqueezyEventSubscriptionPaymentSuccess:
		return s.handleSubscriptionPaymentSuccess(ctx, payload)

	case LemonSqueezyEventSubscriptionPaymentFailed:
		return s.handleSubscriptionPaymentFailed(ctx, payload)

	case LemonSqueezyEventOrderCreated:
		// Order created events can be logged but usually don't require action
		// since subscription_created will be sent separately
		log.Info("Order created", "order_id", payload.Data.Attributes.OrderID)
		return nil

	default:
		log.Warn("Unhandled webhook event", "event", payload.Meta.EventName)
		return nil
	}
}

// handleSubscriptionCreated handles subscription creation
func (s *Service) handleSubscriptionCreated(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	// Get user ID and instance ID from custom data
	userID := payload.Meta.CustomData.UserID
	instanceID := payload.Meta.CustomData.InstanceID

	if userID == "" || instanceID == "" {
		return fmt.Errorf("missing user_id or instance_id in custom_data")
	}

	// Parse trial end date if present
	var trialEndsAt sql.NullTime
	if payload.Data.Attributes.TrialEndsAt != nil && *payload.Data.Attributes.TrialEndsAt != "" {
		t, err := time.Parse(time.RFC3339, *payload.Data.Attributes.TrialEndsAt)
		if err == nil {
			trialEndsAt = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Map Lemon Squeezy status to internal status
	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	// Create subscription record
	_, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              userID,
		InstanceID:          instanceID,
		ProductID:      fmt.Sprintf("%d", payload.Data.Attributes.ProductID),
		CustomerID:     fmt.Sprintf("%d", payload.Data.Attributes.CustomerID),
		SubscriptionID: payload.Data.ID,
		TrialEndsAt:         trialEndsAt,
		Status:              status,
	})

	if err != nil {
		log.Error("Failed to create subscription", "error", err)
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Info("Subscription created",
		"subscription_id", payload.Data.ID,
		"user_id", userID,
		"instance_id", instanceID,
		"status", status)

	return nil
}

// handleSubscriptionUpdated handles subscription updates
func (s *Service) handleSubscriptionUpdated(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              status,
	})

	if err != nil {
		log.Error("Failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Info("Subscription updated", "subscription_id", payload.Data.ID, "status", status)
	return nil
}

// handleSubscriptionCancelled handles subscription cancellation
func (s *Service) handleSubscriptionCancelled(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              "cancelled",
	})

	if err != nil {
		log.Error("Failed to cancel subscription", "error", err)
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	log.Info("Subscription cancelled", "subscription_id", payload.Data.ID)
	return nil
}

// handleSubscriptionResumed handles subscription resumption
func (s *Service) handleSubscriptionResumed(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              status,
	})

	if err != nil {
		log.Error("Failed to resume subscription", "error", err)
		return fmt.Errorf("failed to resume subscription: %w", err)
	}

	log.Info("Subscription resumed", "subscription_id", payload.Data.ID, "status", status)
	return nil
}

// handleSubscriptionExpired handles subscription expiration
func (s *Service) handleSubscriptionExpired(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              "expired",
	})

	if err != nil {
		log.Error("Failed to expire subscription", "error", err)
		return fmt.Errorf("failed to expire subscription: %w", err)
	}

	log.Info("Subscription expired", "subscription_id", payload.Data.ID)
	return nil
}

// handleSubscriptionPaused handles subscription pause
func (s *Service) handleSubscriptionPaused(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              "paused",
	})

	if err != nil {
		log.Error("Failed to pause subscription", "error", err)
		return fmt.Errorf("failed to pause subscription: %w", err)
	}

	log.Info("Subscription paused", "subscription_id", payload.Data.ID)
	return nil
}

// handleSubscriptionUnpaused handles subscription unpause
func (s *Service) handleSubscriptionUnpaused(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              status,
	})

	if err != nil {
		log.Error("Failed to unpause subscription", "error", err)
		return fmt.Errorf("failed to unpause subscription: %w", err)
	}

	log.Info("Subscription unpaused", "subscription_id", payload.Data.ID, "status", status)
	return nil
}

// handleSubscriptionPaymentSuccess handles successful payment
func (s *Service) handleSubscriptionPaymentSuccess(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	// Update subscription status to active if payment succeeded
	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              "active",
	})

	if err != nil {
		log.Error("Failed to update subscription after payment success", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Info("Subscription payment successful", "subscription_id", payload.Data.ID)
	return nil
}

// handleSubscriptionPaymentFailed handles failed payment
func (s *Service) handleSubscriptionPaymentFailed(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := db.New(s.db)

	// Update subscription status to past_due when payment fails
	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:              "past_due",
	})

	if err != nil {
		log.Error("Failed to update subscription after payment failure", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Warn("Subscription payment failed", "subscription_id", payload.Data.ID)
	return nil
}

// mapLemonSqueezyStatus maps Lemon Squeezy status to internal status
func mapLemonSqueezyStatus(lsStatus string) string {
	switch lsStatus {
	case LemonSqueezyStatusOnTrial:
		return "trialing"
	case LemonSqueezyStatusActive:
		return "active"
	case LemonSqueezyStatusPaused:
		return "paused"
	case LemonSqueezyStatusPastDue:
		return "past_due"
	case LemonSqueezyStatusUnpaid:
		return "unpaid"
	case LemonSqueezyStatusCancelled:
		return "cancelled"
	case LemonSqueezyStatusExpired:
		return "expired"
	default:
		return lsStatus
	}
}
