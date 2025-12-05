package provisioning

import (
	"context"
	"fmt"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// List Instances API types
type ListInstancesRequest struct {
	UserID string `json:"user_id"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type ListInstancesResponse struct {
	Instances []*InstanceStatus `json:"instances"`
}

//encore:api private
func (s *Service) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	queries := db.New(s.db)

	var instances []db.Instance
	var err error

	if req.UserID != "" {
		instances, err = queries.ListInstancesByUser(ctx, req.UserID)
	} else {
		instances, err = queries.ListAllInstances(ctx, db.ListAllInstancesParams{
			Limit:  int32(limit),
			Offset: int32(req.Offset),
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	result := make([]*InstanceStatus, len(instances))
	for i, instance := range instances {
		result[i] = &InstanceStatus{
			ID:         instance.ID,
			Status:     instance.Status,
			Domain:     fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain),
			Namespace:  instance.Namespace,
			ServiceURL: fmt.Sprintf("n8n-main.%s.svc.cluster.local", instance.Namespace),
			CreatedAt:  instance.CreatedAt.Time,
		}

		if instance.DeployedAt.Valid {
			result[i].DeployedAt = &instance.DeployedAt.Time
		}
	}

	return &ListInstancesResponse{Instances: result}, nil
}
