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
	Seats      int64  `json:"seats,omitempty"`
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

	if req.Seats <= 0 {
		req.Seats = 1
	}

	checkoutCreate := components.CheckoutCreate{
		// Seats:              polargo.Pointer(req.Seats),
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

	return &CreateCheckoutResponse{
		CheckoutURL: resp.Checkout.URL,
		CheckoutID:  resp.Checkout.ID,
	}, nil
}

type HandleCheckoutCallbackRequest struct {
	CheckoutID string `json:"checkout_id"`
}

type HandleCheckoutCallbackResponse struct {
	SubscriptionID string `json:"subscription_id,omitempty"`
}

//encore:api private
func (s *Service) HandleCheckoutCallback(ctx context.Context, req *HandleCheckoutCallbackRequest) (*HandleCheckoutCallbackResponse, error) {
	// Fetch checkout details
	checkout, err := s.polarClient.Checkouts.Get(ctx, req.CheckoutID)
	if err != nil {
		rlog.Error("Failed to fetch checkout details", "error", err, "checkout_id", req.CheckoutID)
		return nil, fmt.Errorf("failed to fetch checkout details: %w", err)
	}

	rlog.Info("Processing checkout callback",
		"checkout_id", req.CheckoutID,
		"status", checkout.Checkout.Status,
	)

	if checkout.Checkout.Status != components.CheckoutStatusSucceeded {
		return nil, fmt.Errorf("checkout not completed successfully: status=%s", checkout.Checkout.Status)
	}

	// create subscription if not exists
	queries := db.New(s.db)

	subscriptionRow, err := queries.GetSubscriptionByUserID(ctx, *checkout.Checkout.ExternalCustomerID)
	if err != nil && !db.IsNotFoundError(err) {
		rlog.Error("Failed to get subscription by user ID", "error", err, "user_id", *checkout.Checkout.ExternalCustomerID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	var res HandleCheckoutCallbackResponse

	if db.IsNotFoundError(err) {
		// Create new subscription
		sub, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
			UserID:          *checkout.Checkout.ExternalCustomerID,
			PolarCustomerID: *checkout.Checkout.CustomerID,
			// PolarSubscriptionID: *checkout.Checkout.SubscriptionID,
			PolarProductID: *checkout.Checkout.ProductID,
			Status:         StatusActive,
		})
		if err != nil {
			rlog.Error("Failed to create subscription after checkout", "error", err, "user_id", *checkout.Checkout.ExternalCustomerID)
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}

		rlog.Info("Subscription created successfully after checkout", "user_id", *checkout.Checkout.ExternalCustomerID)
		res.SubscriptionID = sub.ID
	} else {
		res.SubscriptionID = subscriptionRow.ID
		// Update existing subscription to active
		err = queries.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscriptionRow.ID,
			Status: StatusActive,
		})
		if err != nil {
			rlog.Error("Failed to update subscription status after checkout", "error", err, "user_id", *checkout.Checkout.ExternalCustomerID)
			return nil, fmt.Errorf("failed to update subscription status: %w", err)
		}

		rlog.Info("Subscription status updated to active after checkout", "user_id", *checkout.Checkout.ExternalCustomerID)
	}

	return &res, nil
}
