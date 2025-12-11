package subscription

import (
	"context"
	"fmt"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
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

type HandleCheckoutCallbackRequest struct {
	CheckoutID string `json:"checkout_id"`
}

type HandleCheckoutCallbackResponse struct {
	SubscriptionID string `json:"subscription_id,omitempty"`
}

//encore:api private
func (s *Service) HandleCheckoutCallback(ctx context.Context, req *HandleCheckoutCallbackRequest) (*HandleCheckoutCallbackResponse, error) {
	queries := db.New(s.db)

	// Fetch checkout session from database
	checkoutSession, err := queries.GetCheckoutSessionByPolarID(ctx, req.CheckoutID)
	if err != nil {
		rlog.Error("Failed to fetch checkout session from database", "error", err, "checkout_id", req.CheckoutID)
		return nil, fmt.Errorf("failed to fetch checkout session: %w", err)
	}

	// Fetch checkout details from Polar
	checkout, err := s.polarClient.Checkouts.Get(ctx, req.CheckoutID)
	if err != nil {
		rlog.Error("Failed to fetch checkout details from Polar", "error", err, "checkout_id", req.CheckoutID)
		return nil, fmt.Errorf("failed to fetch checkout details: %w", err)
	}

	if checkout.Checkout.Status != components.CheckoutStatusSucceeded {
		return nil, fmt.Errorf("checkout not completed successfully: status=%s", checkout.Checkout.Status)
	}

	userID := *checkout.Checkout.ExternalCustomerID

	rlog.Info("Processing checkout callback",
		"checkout_id", req.CheckoutID,
		"subdomain", checkoutSession.Subdomain,
		"user_id", userID,
		"status", checkout.Checkout.Status,
	)

	// Create and deploy the instance
	provisionResp, err := provisioning.CreateInstance(ctx, &provisioning.CreateInstanceRequest{
		UserID:    userID,
		Subdomain: checkoutSession.Subdomain,
		DeployNow: true,
	})
	if err != nil {
		rlog.Error("Failed to create and deploy instance", "error", err, "subdomain", checkoutSession.Subdomain)
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	rlog.Info("Instance created and deployed successfully",
		"instance_id", provisionResp.InstanceID,
		"user_id", userID,
		"subdomain", checkoutSession.Subdomain,
	)

	// Validate Polar checkout data
	if checkout.Checkout.CustomerID == nil || checkout.Checkout.SubscriptionID == nil || checkout.Checkout.ProductID == nil {
		rlog.Error("Missing Polar data in checkout",
			"checkout_id", req.CheckoutID,
			"customer_id", checkout.Checkout.CustomerID,
			"subscription_id", checkout.Checkout.SubscriptionID,
			"product_id", checkout.Checkout.ProductID,
		)
		return nil, fmt.Errorf("missing required Polar data in checkout")
	}

	// Create active subscription for this instance
	sub, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
		UserID:              userID,
		InstanceID:          provisionResp.InstanceID,
		PolarCustomerID:     *checkout.Checkout.CustomerID,
		PolarSubscriptionID: *checkout.Checkout.SubscriptionID,
		PolarProductID:      *checkout.Checkout.ProductID,
		Status:              StatusActive,
	})
	if err != nil {
		rlog.Error("Failed to create subscription after checkout", "error", err, "instance_id", provisionResp.InstanceID)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Update checkout session status to completed
	err = queries.UpdateCheckoutSessionStatus(ctx, db.UpdateCheckoutSessionStatusParams{
		ID:     checkoutSession.ID,
		Status: "completed",
	})
	if err != nil {
		rlog.Error("Failed to update checkout session status", "error", err, "checkout_session_id", checkoutSession.ID)
		// Not a critical error, continue
	}

	rlog.Info("Subscription created successfully after checkout",
		"instance_id", provisionResp.InstanceID,
		"subscription_id", sub.ID,
	)

	return &HandleCheckoutCallbackResponse{
		SubscriptionID: sub.ID,
	}, nil
}
