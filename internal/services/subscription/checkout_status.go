package subscription

import (
	"context"
	"fmt"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// GetCheckoutStatusRequest represents the request to get checkout status
type GetCheckoutStatusRequest struct {
	CheckoutID string `json:"checkout_id"`
}

// GetCheckoutStatusResponse represents the checkout status response
type GetCheckoutStatusResponse struct {
	Status    string `json:"status"`     // pending, completed, failed
	Subdomain string `json:"subdomain"`  // subdomain from checkout session
	UserID    string `json:"user_id"`    // user_id from checkout session
}

// GetCheckoutStatus returns the current status of a checkout session
//
//encore:api private
func (s *Service) GetCheckoutStatus(ctx context.Context, req *GetCheckoutStatusRequest) (*GetCheckoutStatusResponse, error) {
	if req.CheckoutID == "" {
		return nil, fmt.Errorf("checkout_id is required")
	}

	queries := db.New(s.db)

	// Get checkout session
	checkoutSession, err := queries.GetCheckoutSessionByPolarID(ctx, req.CheckoutID)
	if err != nil {
		rlog.Error("Failed to get checkout session", "error", err, "checkout_id", req.CheckoutID)
		return nil, fmt.Errorf("checkout session not found: %w", err)
	}

	return &GetCheckoutStatusResponse{
		Status:    checkoutSession.Status,
		Subdomain: checkoutSession.Subdomain,
		UserID:    checkoutSession.UserID,
	}, nil
}
