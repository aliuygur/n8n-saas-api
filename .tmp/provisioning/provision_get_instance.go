package provisioning

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Get Instance API types
type GetInstanceRequest struct {
	InstanceID string `json:"instance_id"`
}

type Instance struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Status     string     `json:"status"`
	SubDomain  string     `json:"sub_domain"`
	Namespace  string     `json:"namespace"`
	CreatedAt  time.Time  `json:"created_at"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
}

func (i *Instance) GetInstanceURL() string {
	return fmt.Sprintf("https://%s.instol.cloud", i.SubDomain)
}

func (i *Instance) GetServiceURL() string {
	return fmt.Sprintf("n8n-main.%s.svc.cluster.local", i.Namespace)
}

//encore:api private
func (s *Service) GetInstance(ctx context.Context, req *GetInstanceRequest) (*Instance, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, req.InstanceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	response := &Instance{
		ID:        instance.ID,
		UserID:    instance.UserID,
		Status:    instance.Status,
		SubDomain: instance.Subdomain,
		Namespace: instance.Namespace,
		CreatedAt: instance.CreatedAt.Time,
	}

	if instance.DeployedAt.Valid {
		response.DeployedAt = &instance.DeployedAt.Time
	}

	return response, nil
}
