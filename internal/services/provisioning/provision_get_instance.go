package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Get Instance API types
type GetInstanceRequest struct {
	InstanceID int `json:"instance_id"`
}

type InstanceStatus struct {
	ID         int        `json:"id"`
	Status     string     `json:"status"`
	Domain     string     `json:"domain"`
	Namespace  string     `json:"namespace"`
	ServiceURL string     `json:"service_url"`
	CreatedAt  time.Time  `json:"created_at"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
	Details    string     `json:"details,omitempty"` // JSON string instead of interface{}
}

//encore:api private
func (s *Service) GetInstance(ctx context.Context, req *GetInstanceRequest) (*InstanceStatus, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	response := &InstanceStatus{
		ID:         int(instance.ID),
		Status:     instance.Status,
		Domain:     fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain),
		Namespace:  instance.Namespace,
		ServiceURL: fmt.Sprintf("n8n-main.%s.svc.cluster.local", instance.Namespace),
		CreatedAt:  instance.CreatedAt.Time,
	}

	if instance.DeployedAt.Valid {
		response.DeployedAt = &instance.DeployedAt.Time
	}

	// Get live status from Kubernetes if deployed
	if instance.Status == "deployed" && instance.Namespace != "" {
		if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err == nil {
			// TODO: Implement GetInstanceStatus method for SQLite-based architecture
			basicDetails := map[string]interface{}{
				"namespace":    instance.Namespace,
				"architecture": "sqlite_isolated",
				"status":       "running",
			}
			if detailsJSON, err := json.Marshal(basicDetails); err == nil {
				response.Details = string(detailsJSON)
			}
		}
	}

	return response, nil
}
