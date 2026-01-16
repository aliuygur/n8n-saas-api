package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/pkg/lemonsqueezy"
	"github.com/jackc/pgx/v5/pgtype"
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
			StoreID               int               `json:"store_id"`
			CustomerID            int               `json:"customer_id"`
			OrderID               int               `json:"order_id"`
			OrderItemID           int               `json:"order_item_id"`
			ProductID             int               `json:"product_id"`
			VariantID             int               `json:"variant_id"`
			ProductName           string            `json:"product_name"`
			VariantName           string            `json:"variant_name"`
			UserName              string            `json:"user_name"`
			UserEmail             string            `json:"user_email"`
			Status                string            `json:"status"`
			StatusFormatted       string            `json:"status_formatted"`
			CardBrand             string            `json:"card_brand"`
			CardLastFour          string            `json:"card_last_four"`
			Pause                 *PauseInfo        `json:"pause"`
			Cancelled             bool              `json:"cancelled"`
			TrialEndsAt           *string           `json:"trial_ends_at"`
			BillingAnchor         int               `json:"billing_anchor"`
			FirstSubscriptionItem *lemonsqueezy.SubscriptionItem `json:"first_subscription_item"`
			Urls                  struct {
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
	UserID string `json:"user_id"`
}

// PauseInfo represents pause information for a subscription
type PauseInfo struct {
	Mode      string `json:"mode"`
	ResumesAt string `json:"resumes_at"`
}

// VerifyLemonSqueezySignature verifies the webhook signature
func (s *Service) VerifyLemonSqueezySignature(payload []byte, signature string) bool {
	return s.lemonsqueezy.VerifyWebhookSignature(payload, signature)
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
	queries := s.getDB()

	// Get user ID from custom data
	userID := payload.Meta.CustomData.UserID

	if userID == "" {
		return fmt.Errorf("missing user_id in custom_data")
	}

	// Check if user has a subscription record
	existingSub, err := queries.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if db.IsNotFoundError(err) {
			log.Error("User has no subscription record", "user_id", userID)
			return fmt.Errorf("user %s has no subscription record", userID)
		}
		log.Error("Failed to get user subscription", "error", err)
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	// Parse trial end date if present
	var trialEndsAt pgtype.Timestamp
	if payload.Data.Attributes.TrialEndsAt != nil && *payload.Data.Attributes.TrialEndsAt != "" {
		t, err := time.Parse(time.RFC3339, *payload.Data.Attributes.TrialEndsAt)
		if err == nil {
			trialEndsAt = pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	// Map Lemon Squeezy status to internal status
	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	// Get quantity from first subscription item
	quantity := int32(1)
	if payload.Data.Attributes.FirstSubscriptionItem != nil {
		quantity = int32(payload.Data.Attributes.FirstSubscriptionItem.Quantity)
	}

	// Update existing subscription with provider details
	err = queries.UpdateSubscriptionByUserID(ctx, db.UpdateSubscriptionByUserIDParams{
		UserID:         userID,
		ProductID:      fmt.Sprintf("%d", payload.Data.Attributes.ProductID),
		VariantID:      fmt.Sprintf("%d", payload.Data.Attributes.VariantID),
		CustomerID:     fmt.Sprintf("%d", payload.Data.Attributes.CustomerID),
		SubscriptionID: payload.Data.ID,
		Status:         status,
		TrialEndsAt:    trialEndsAt,
		Quantity:       quantity,
	})

	if err != nil {
		log.Error("Failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Info("Subscription updated from webhook",
		"subscription_id", payload.Data.ID,
		"user_id", userID,
		"existing_sub_id", existingSub.ID,
		"quantity", quantity,
		"status", status)

	return nil
}

// handleSubscriptionUpdated handles subscription updates
func (s *Service) handleSubscriptionUpdated(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := s.getDB()

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         status,
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
	queries := s.getDB()

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         SubscriptionStatusCanceled,
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
	queries := s.getDB()

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         status,
	})

	if err != nil {
		log.Error("Failed to resume subscription", "error", err)
		return fmt.Errorf("failed to resume subscription: %w", err)
	}

	log.Info("Subscription resumed", "subscription_id", payload.Data.ID, "status", status)
	return nil
}

// handleSubscriptionExpired handles subscription expiration.
// When a subscription expires (e.g., after cancellation period ends),
// all user instances are deleted and the subscription status is updated.
func (s *Service) handleSubscriptionExpired(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := s.getDB()

	// Get subscription to find the user
	sub, err := queries.GetSubscriptionByProviderID(ctx, payload.Data.ID)
	if err != nil {
		log.Error("failed to get subscription", "error", err)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Delete all instances for this user
	if err := s.DeleteAllUserInstances(ctx, sub.UserID); err != nil {
		log.Error("failed to delete user instances", "user_id", sub.UserID, "error", err)
		// Continue to update status even if deletion partially failed
	}

	// Update subscription status
	err = queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         SubscriptionStatusExpired,
	})
	if err != nil {
		log.Error("failed to expire subscription", "error", err)
		return fmt.Errorf("failed to expire subscription: %w", err)
	}

	log.Info("subscription expired and instances deleted",
		"subscription_id", payload.Data.ID,
		"user_id", sub.UserID)
	return nil
}

// handleSubscriptionPaused handles subscription pause
func (s *Service) handleSubscriptionPaused(ctx context.Context, payload *LemonSqueezyWebhookPayload) error {
	log := appctx.GetLogger(ctx)
	queries := s.getDB()

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         SubscriptionStatusPaused,
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
	queries := s.getDB()

	status := mapLemonSqueezyStatus(payload.Data.Attributes.Status)

	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         status,
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
	queries := s.getDB()

	// Update subscription status to active if payment succeeded
	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         SubscriptionStatusActive,
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
	queries := s.getDB()

	// Update subscription status to past_due when payment fails
	err := queries.UpdateSubscriptionStatusByProviderID(ctx, db.UpdateSubscriptionStatusByProviderIDParams{
		SubscriptionID: payload.Data.ID,
		Status:         SubscriptionStatusPastDue,
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
		return SubscriptionStatusTrialing
	case LemonSqueezyStatusActive:
		return SubscriptionStatusActive
	case LemonSqueezyStatusPaused:
		return SubscriptionStatusPaused
	case LemonSqueezyStatusPastDue:
		return SubscriptionStatusPastDue
	case LemonSqueezyStatusUnpaid:
		return SubscriptionStatusUnpaid
	case LemonSqueezyStatusCancelled:
		return SubscriptionStatusCanceled
	case LemonSqueezyStatusExpired:
		return SubscriptionStatusExpired
	default:
		return lsStatus
	}
}

// CreateUpgradeCheckoutURL creates a LemonSqueezy checkout URL for upgrading from trial to paid
func (s *Service) CreateUpgradeCheckoutURL(ctx context.Context, userID string) (string, error) {
	// Get user details for email
	queries := s.getDB()
	user, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return fmt.Sprintf(
		"https://ranx.lemonsqueezy.com/buy/%s?checkout[email]=%s&checkout[custom][user_id]=%s",
		s.config.LemonSqueezy.VariantID,
		user.Email,
		user.ID,
	), nil
}
