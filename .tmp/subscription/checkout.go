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
	InstanceID string `json:"instance_id"`
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
		Products:           []string{secrets.POLAR_PRODUCT_ID},
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
		InstanceID:      req.InstanceID,
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

type CheckoutSession struct {
	CheckoutID string `json:"checkout_id"`
	UserID     string `json:"user_id"`
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`
	Subdomain  string `json:"subdomain"`
}

type GetCheckoutSesionByPolarIDRequest struct {
	PolarCheckoutID string `json:"polar_checkout_id"`
}

type GetCheckoutResponse struct {
	Session CheckoutSession `json:"session"`
}

// GetCheckout retrieves a checkout session by its Polar checkout ID
//
//encore:api private
func (s *Service) GetCheckoutSessionByPolarID(ctx context.Context, req *GetCheckoutSesionByPolarIDRequest) (*GetCheckoutResponse, error) {
	queries := db.New(s.db)
	checkoutSession, err := queries.GetCheckoutSessionByPolarID(ctx, req.PolarCheckoutID)
	if err != nil {
		rlog.Error("Failed to get checkout session from database", "error", err, "polar_checkout_id", req.PolarCheckoutID)
		return nil, fmt.Errorf("failed to get checkout session: %w", err)
	}

	return &GetCheckoutResponse{
		Session: CheckoutSession{
			CheckoutID: checkoutSession.PolarCheckoutID,
			UserID:     checkoutSession.UserID,
			InstanceID: checkoutSession.InstanceID,
			Status:     checkoutSession.Status,
			Subdomain:  checkoutSession.Subdomain,
		},
	}, nil
}
