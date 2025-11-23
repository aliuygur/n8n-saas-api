package provisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Get Instance Deployments API types
type GetInstanceDeploymentsRequest struct {
	InstanceID int `json:"instance_id"`
	Limit      int `json:"limit,omitempty"`
	Offset     int `json:"offset,omitempty"`
}

type GetInstanceDeploymentsResponse struct {
	Deployments []*DeploymentInfo `json:"deployments"`
}

type DeploymentInfo struct {
	ID           int        `json:"id"`
	InstanceID   int        `json:"instance_id"`
	Operation    string     `json:"operation"`
	Status       string     `json:"status"`
	Details      string     `json:"details,omitempty"` // JSON string instead of interface{}
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

//encore:api private
func (s *Service) GetInstanceDeployments(ctx context.Context, req *GetInstanceDeploymentsRequest) (*GetInstanceDeploymentsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	queries := db.New(s.db)

	deployments, err := queries.ListDeploymentsByInstance(ctx, db.ListDeploymentsByInstanceParams{
		InstanceID: int32(req.InstanceID),
		Limit:      int32(limit),
		Offset:     int32(req.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	result := make([]*DeploymentInfo, len(deployments))
	for i, deployment := range deployments {
		result[i] = &DeploymentInfo{
			ID:         int(deployment.ID),
			InstanceID: int(deployment.InstanceID),
			Operation:  deployment.Operation,
			Status:     deployment.Status,
			StartedAt:  deployment.StartedAt.Time,
		}

		if deployment.ErrorMessage != "" {
			result[i].ErrorMessage = &deployment.ErrorMessage
		}

		if deployment.CompletedAt.Valid {
			result[i].CompletedAt = &deployment.CompletedAt.Time
		}

		// Parse details JSON if available
		if len(deployment.Details) > 0 {
			result[i].Details = string(deployment.Details)
		}
	}

	return &GetInstanceDeploymentsResponse{Deployments: result}, nil
}
