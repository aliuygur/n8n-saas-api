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
	InstanceID string `json:"instance_id"`
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
		// Store instance_id in metadata so we can deploy it after payment
		Metadata: map[string]components.CheckoutCreateMetadata{
			"instance_id": components.CreateCheckoutCreateMetadataStr(req.InstanceID),
		},
	}

	if req.ReturnURL != "" {
		checkoutCreate.ReturnURL = polargo.Pointer(req.ReturnURL)
	}

	rlog.Info("Creating Polar checkout session",
		"user_id", req.UserID,
		"instance_id", req.InstanceID,
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

	if checkout.Checkout.Status != components.CheckoutStatusSucceeded {
		return nil, fmt.Errorf("checkout not completed successfully: status=%s", checkout.Checkout.Status)
	}

	// Extract instance_id from checkout metadata
	instanceID := ""
	if checkout.Checkout.Metadata != nil {
		if metadata, ok := checkout.Checkout.Metadata["instance_id"]; ok {
			if metadata.Str != nil {
				instanceID = *metadata.Str
			}
		}
	}

	if instanceID == "" {
		rlog.Error("instance_id not found in checkout metadata", "checkout_id", req.CheckoutID)
		return nil, fmt.Errorf("instance_id not found in checkout metadata")
	}

	userID := *checkout.Checkout.ExternalCustomerID

	rlog.Info("Processing checkout callback",
		"checkout_id", req.CheckoutID,
		"instance_id", instanceID,
		"user_id", userID,
		"status", checkout.Checkout.Status,
	)

	// Get the pending instance by ID
	queries := db.New(s.db)
	instance, err := queries.GetInstance(ctx, instanceID)
	if err != nil {
		rlog.Error("Failed to get instance", "error", err, "instance_id", instanceID)
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Deploy the pending instance
	provisionResp, err := provisioning.DeployPendingInstance(ctx, &provisioning.DeployPendingInstanceRequest{
		Subdomain: instance.Subdomain,
	})
	if err != nil {
		rlog.Error("Failed to deploy pending instance", "error", err, "instance_id", instanceID)
		return nil, fmt.Errorf("failed to deploy instance: %w", err)
	}

	rlog.Info("Instance provisioned successfully",
		"instance_id", provisionResp.InstanceID,
		"user_id", userID,
	)

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

	rlog.Info("Subscription created successfully after checkout",
		"instance_id", provisionResp.InstanceID,
		"subscription_id", sub.ID,
	)

	return &HandleCheckoutCallbackResponse{
		SubscriptionID: sub.ID,
	}, nil
}
