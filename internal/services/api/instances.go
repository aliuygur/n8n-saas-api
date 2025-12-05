package api

import (
	"context"
	"fmt"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
	"github.com/aliuygur/n8n-saas-api/internal/services/subscription"
)

// CreateInstance creates a new n8n instance for the authenticated user

type CreateInstanceRequest struct {
	Subdomain string `json:"subdomain"`
}

type CreateInstanceResponse struct {
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`
	Domain     string `json:"domain"`
}

// CreateInstance creates a new n8n instance for the authenticated user
//
//encore:api auth method=POST path=/me/instances
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	// Get user ID from auth context
	uid, ok := auth.UserID()
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}
	userID := string(uid)

	rlog.Debug("Creating instance", "user_id", userID, "subdomain", req.Subdomain)

	// Get user to retrieve user ID (UUID)
	queries := db.New(s.db)
	user, err := queries.GetUserByEmail(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

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
			// This will return appropriate error messages for trial limits, expired subscriptions, etc.
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
		// Preserve the error as-is to maintain error codes
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

	return &CreateInstanceResponse{
		InstanceID: provResp.InstanceID,
		Status:     provResp.Status,
		Domain:     provResp.Domain,
	}, nil
}

// GetInstance retrieves a specific instance by ID

type InstanceResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Domain     string `json:"domain"`
	Namespace  string `json:"namespace"`
	ServiceURL string `json:"service_url"`
	CreatedAt  string `json:"created_at"`
	DeployedAt string `json:"deployed_at,omitempty"`
	Details    string `json:"details,omitempty"`
}

//encore:api auth method=GET path=/me/instances/:id
func (s *Service) GetInstance(ctx context.Context, id string) (*InstanceResponse, error) {
	// Get user ID from auth context
	uid, ok := auth.UserID()
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}
	userID := string(uid)

	rlog.Debug("Getting instance", "user_id", userID, "instance_id", id)

	// Call provisioning service
	provResp, err := provisioning.GetInstance(ctx, &provisioning.GetInstanceRequest{
		InstanceID: id,
	})
	if err != nil {
		rlog.Error("Failed to get instance", "error", err)
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Verify the instance belongs to the authenticated user
	if provResp.UserID != userID {
		rlog.Warn("Unauthorized instance access attempt", "user_id", userID, "instance_id", id, "owner_id", provResp.UserID)
		return nil, &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "you do not have permission to access this instance",
		}
	}

	// Convert timestamps to strings
	createdAt := provResp.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	deployedAt := ""
	if provResp.DeployedAt != nil {
		deployedAt = provResp.DeployedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &InstanceResponse{
		ID:         provResp.ID,
		Status:     provResp.Status,
		Domain:     provResp.Domain,
		Namespace:  provResp.Namespace,
		ServiceURL: provResp.ServiceURL,
		CreatedAt:  createdAt,
		DeployedAt: deployedAt,
		Details:    provResp.Details,
	}, nil
}

// ListInstances retrieves all instances for the authenticated user

type ListInstancesRequest struct {
	Limit  int `query:"limit,omitempty"`
	Offset int `query:"offset,omitempty"`
}

type ListInstancesResponse struct {
	Instances []*InstanceResponse `json:"instances"`
}

//encore:api auth method=GET path=/me/instances
func (s *Service) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponse, error) {
	// Get user ID from auth context
	uid, ok := auth.UserID()
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}
	userID := string(uid)

	rlog.Debug("Listing instances", "user_id", userID, "limit", req.Limit, "offset", req.Offset)

	// Call provisioning service
	provResp, err := provisioning.ListInstances(ctx, &provisioning.ListInstancesRequest{
		UserID: userID,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		rlog.Error("Failed to list instances", "error", err)
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	// Convert response
	instances := make([]*InstanceResponse, len(provResp.Instances))
	for i, inst := range provResp.Instances {
		createdAt := inst.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		deployedAt := ""
		if inst.DeployedAt != nil {
			deployedAt = inst.DeployedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		instances[i] = &InstanceResponse{
			ID:         inst.ID,
			Status:     inst.Status,
			Domain:     inst.Domain,
			Namespace:  inst.Namespace,
			ServiceURL: inst.ServiceURL,
			CreatedAt:  createdAt,
			DeployedAt: deployedAt,
			Details:    inst.Details,
		}
	}

	return &ListInstancesResponse{
		Instances: instances,
	}, nil
}

// DeleteInstance deletes an existing instance

type DeleteInstanceResponse struct {
	Message string `json:"message"`
}

//encore:api auth method=DELETE path=/me/instances/:id
func (s *Service) DeleteInstance(ctx context.Context, id string) (*DeleteInstanceResponse, error) {
	// Get user ID from auth context
	uid, ok := auth.UserID()
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}
	userID := string(uid)

	rlog.Debug("Deleting instance", "user_id", userID, "instance_id", id)

	// Get user to retrieve user ID (UUID)
	queries := db.New(s.db)
	user, err := queries.GetUserByEmail(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// First, verify the instance belongs to this user
	instance, err := provisioning.GetInstance(ctx, &provisioning.GetInstanceRequest{
		InstanceID: id,
	})
	if err != nil {
		rlog.Error("Failed to get instance", "error", err)
		return nil, fmt.Errorf("failed to verify instance ownership: %w", err)
	}

	// Verify the instance belongs to the authenticated user
	if instance.UserID != user.ID {
		rlog.Warn("Unauthorized instance deletion attempt", "user_id", user.ID, "instance_id", id, "owner_id", instance.UserID)
		return nil, &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "you do not have permission to delete this instance",
		}
	}

	// Call provisioning service to delete
	err = provisioning.DeleteInstance(ctx, &provisioning.DeleteInstanceRequest{
		InstanceID: instance.ID,
	})
	if err != nil {
		rlog.Error("Failed to delete instance", "error", err)
		return nil, fmt.Errorf("failed to delete instance: %w", err)
	}

	// Decrement instance count in subscription
	err = subscription.DecrementInstance(ctx, &subscription.DecrementInstanceRequest{
		UserID: user.ID,
	})
	if err != nil {
		rlog.Error("Failed to decrement instance count", "error", err)
		// Don't fail the deletion, just log the error
	}

	return &DeleteInstanceResponse{
		Message: "Instance successfully deleted",
	}, nil
}
