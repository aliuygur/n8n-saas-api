package subscription

import (
	"context"
	"fmt"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

// CreateCheckoutRequest represents the request to create a checkout session
type CreateCheckoutRequest struct {
	UserID     string `json:"user_id"`
	Subdomain  string `json:"subdomain"`
	UserEmail  string `json:"user_email,omitempty"`
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

	checkoutCreate := components.CheckoutCreate{
		Products:           []string{secrets.PolarProductID},
		ExternalCustomerID: polargo.Pointer(req.UserID),
		CustomerEmail:      polargo.Pointer(req.UserEmail),
		SuccessURL:         polargo.Pointer(req.SuccessURL),
	}

	if req.ReturnURL != "" {
		checkoutCreate.ReturnURL = polargo.Pointer(req.ReturnURL)
	}

	rlog.Info("Creating Polar checkout session",
		"user_id", req.UserID,
		"subdomain", req.Subdomain,
		"email", req.UserEmail,
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

	// Store checkout session in database
	queries := db.New(s.db)
	checkoutSession, err := queries.CreateCheckoutSession(ctx, db.CreateCheckoutSessionParams{
		UserID:          req.UserID,
		PolarCheckoutID: resp.Checkout.ID,
		Subdomain:       req.Subdomain,
		UserEmail:       req.UserEmail,
		SuccessUrl:      req.SuccessURL,
		ReturnUrl:       req.ReturnURL,
		Status:          "pending",
	})
	if err != nil {
		rlog.Error("Failed to store checkout session in database", "error", err)
		return nil, fmt.Errorf("failed to store checkout session: %w", err)
	}

	rlog.Info("Checkout session stored in database",
		"checkout_session_id", checkoutSession.ID,
		"polar_checkout_id", checkoutSession.PolarCheckoutID,
	)

	return &CreateCheckoutResponse{
		CheckoutURL: resp.Checkout.URL,
		CheckoutID:  resp.Checkout.ID,
	}, nil
}
