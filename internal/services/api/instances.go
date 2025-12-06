package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"encore.dev"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/auth"
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
	"github.com/aliuygur/n8n-saas-api/internal/services/subscription"
)

// CreateInstanceRequest represents the request to create a new instance
type CreateInstanceRequest struct {
	Subdomain string `json:"subdomain"`
}

// CreateInstanceResponse represents the response from creating an instance
type CreateInstanceResponse struct {
	ID          string `json:"id"`
	InstanceURL string `json:"instance_url"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// CheckSubdomainRequest represents the request to check subdomain availability
type CheckSubdomainRequest struct {
	Subdomain string `json:"subdomain"`
}

// CheckSubdomainResponse represents the response from checking subdomain availability
type CheckSubdomainResponse struct {
	Available       bool   `json:"available"`
	Message         string `json:"message"`
	ValidationError bool   `json:"validation_error"` // true if error is due to validation, false if subdomain exists
}

// CheckSubdomainAvailability checks if a subdomain is available
//
//encore:api auth method=POST path=/api/instances/check-subdomain
func (s *Service) CheckSubdomainAvailability(ctx context.Context, req *CheckSubdomainRequest) (*CheckSubdomainResponse, error) {
	rlog.Debug("Checking subdomain availability", "subdomain", req.Subdomain)

	// Validate subdomain using provisioning service's validation
	validationResp, err := provisioning.ValidateSubdomain(ctx, &provisioning.ValidateSubdomainRequest{
		Subdomain: req.Subdomain,
	})
	if err != nil {
		rlog.Error("Failed to validate subdomain", "error", err)
		return nil, fmt.Errorf("failed to validate subdomain: %w", err)
	}

	if !validationResp.Valid {
		return &CheckSubdomainResponse{
			Available:       false,
			Message:         validationResp.ErrorMessage,
			ValidationError: true,
		}, nil
	}

	// Check if subdomain already exists
	exists, err := provisioning.CheckSubdomainExists(ctx, &provisioning.CheckSubdomainExistsRequest{
		Subdomain: req.Subdomain,
	})
	if err != nil {
		rlog.Error("Failed to check subdomain", "error", err)
		return nil, fmt.Errorf("failed to check subdomain availability: %w", err)
	}

	if exists.Exists {
		return &CheckSubdomainResponse{
			Available:       false,
			Message:         "This subdomain is already taken",
			ValidationError: false,
		}, nil
	}

	return &CheckSubdomainResponse{
		Available:       true,
		Message:         "Subdomain is available",
		ValidationError: false,
	}, nil
}

// CreateInstance creates a new n8n instance
//
//encore:api auth method=POST path=/api/instances
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	user := auth.MustGetUser()

	rlog.Debug("Creating instance", "user_id", user.ID, "subdomain", req.Subdomain)

	// Check subscription status
	subStatus, err := subscription.GetSubscriptionStatus(ctx, &subscription.GetSubscriptionStatusRequest{
		UserID: user.ID,
	})
	if err != nil {
		rlog.Error("Failed to get subscription status", "error", err)
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	// If user has no subscription, start trial for first instance
	if subStatus.Status == "none" {
		_, err := subscription.StartTrial(ctx, &subscription.StartTrialRequest{
			UserID: user.ID,
		})
		if err != nil {
			rlog.Error("Failed to start trial", "error", err)
			return nil, fmt.Errorf("failed to start trial: %w", err)
		}
		rlog.Info("Trial started for user", "user_id", user.ID)
	} else {
		// Validate if user can create another instance
		_, err := subscription.ValidateInstanceCreation(ctx, &subscription.ValidateInstanceCreationRequest{
			UserID: user.ID,
		})
		if err != nil {
			rlog.Error("Error validating instance creation", "error", err)
			return nil, err
		}
	}

	// Call provisioning service
	provResp, err := provisioning.CreateInstance(ctx, &provisioning.CreateInstanceRequest{
		UserID:    user.ID,
		Subdomain: req.Subdomain,
	})
	if err != nil {
		rlog.Error("Failed to create instance", "error", err)
		return nil, err
	}

	// Increment instance count if not first instance
	if subStatus.Status != "none" {
		err = subscription.IncrementInstance(ctx, &subscription.IncrementInstanceRequest{
			UserID: user.ID,
		})
		if err != nil {
			rlog.Error("Failed to increment instance count", "error", err)
			// Don't fail instance creation, just log the error
		}
	}

	rlog.Info("Instance created successfully", "subdomain", req.Subdomain, "instance_id", provResp.InstanceID, "domain", provResp.Domain)

	return &CreateInstanceResponse{
		ID:          provResp.InstanceID,
		InstanceURL: provResp.Domain,
		Status:      provResp.Status,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// Instance represents an n8n instance
type Instance struct {
	ID          string `json:"id"`
	InstanceURL string `json:"instance_url"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ListInstancesResponse represents the response from listing instances
type ListInstancesResponse struct {
	Instances []Instance `json:"instances"`
}

// ListInstances returns all instances for the authenticated user
//
//encore:api auth method=GET path=/api/instances
func (s *Service) ListInstances(ctx context.Context) (*ListInstancesResponse, error) {
	user := auth.MustGetUser()

	rlog.Debug("Listing instances", "user_id", user.ID)

	// Call provisioning service
	provResp, err := provisioning.ListInstances(ctx, &provisioning.ListInstancesRequest{
		UserID: user.ID,
	})
	if err != nil {
		rlog.Error("Failed to list instances", "error", err)
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	// Convert to API response format
	instances := make([]Instance, len(provResp.Instances))
	for i, inst := range provResp.Instances {
		instances[i] = Instance{
			ID:          inst.ID,
			InstanceURL: inst.Domain,
			Status:      inst.Status,
			CreatedAt:   inst.CreatedAt.Format(time.RFC3339),
		}
	}

	return &ListInstancesResponse{
		Instances: instances,
	}, nil
}

// GetInstanceResponse represents a single instance response
type GetInstanceResponse struct {
	ID          string `json:"id"`
	InstanceURL string `json:"instance_url"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// GetInstance returns a specific instance
//
//encore:api auth method=GET path=/api/instances/:id
func (s *Service) GetInstance(ctx context.Context, id string) (*GetInstanceResponse, error) {
	user := auth.MustGetUser()

	rlog.Debug("Getting instance", "user_id", user.ID, "instance_id", id)

	// Call provisioning service
	instance, err := provisioning.GetInstance(ctx, &provisioning.GetInstanceRequest{
		InstanceID: id,
	})
	if err != nil {
		rlog.Error("Failed to get instance", "error", err)
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Verify ownership
	if instance.UserID != user.ID {
		rlog.Warn("Unauthorized instance access attempt", "user_id", user.ID, "instance_id", id, "owner_id", instance.UserID)
		return nil, fmt.Errorf("you do not have permission to access this instance")
	}

	return &GetInstanceResponse{
		ID:          instance.ID,
		InstanceURL: instance.Domain,
		Status:      instance.Status,
		CreatedAt:   instance.CreatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteInstance deletes an instance
//
//encore:api auth raw method=DELETE path=/api/instances/:id
func (s *Service) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := encore.CurrentRequest().PathParams.Get("id")

	user := auth.MustGetUser()

	rlog.Debug("Deleting instance", "user_id", user.ID, "instance_id", instanceID)

	// Validate instance ID
	if instanceID == "" {
		rlog.Error("Empty instance ID received")
		http.Error(w, "instance ID is required", http.StatusBadRequest)
		return
	}

	// First, verify the instance belongs to this user
	instance, err := provisioning.GetInstance(ctx, &provisioning.GetInstanceRequest{
		InstanceID: instanceID,
	})
	if err != nil {
		rlog.Error("Failed to get instance", "error", err)
		http.Error(w, fmt.Sprintf("failed to verify instance ownership: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify ownership
	if instance.UserID != user.ID {
		rlog.Warn("Unauthorized instance deletion attempt", "user_id", user.ID, "instance_id", instanceID, "owner_id", instance.UserID)
		http.Error(w, "you do not have permission to delete this instance", http.StatusForbidden)
		return
	}

	// Call provisioning service to delete
	err = provisioning.DeleteInstance(ctx, &provisioning.DeleteInstanceRequest{
		InstanceID: instance.ID,
	})
	if err != nil {
		rlog.Error("Failed to delete instance", "error", err)
		http.Error(w, fmt.Sprintf("failed to delete instance: %v", err), http.StatusInternalServerError)
		return
	}

	// Decrement instance count in subscription
	err = subscription.DecrementInstance(ctx, &subscription.DecrementInstanceRequest{
		UserID: user.ID,
	})
	if err != nil {
		rlog.Error("Failed to decrement instance count", "error", err)
		// Don't fail the deletion, just log the error
	}

	rlog.Info("Instance deleted successfully", "instance_id", instanceID, "user_id", user.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
