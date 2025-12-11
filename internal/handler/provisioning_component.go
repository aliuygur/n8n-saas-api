package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
)

// getInstanceForComponent gets an instance and converts it to provisioning.Instance for use with components
func (h *Handler) getInstanceForComponent(ctx context.Context, instanceID string) (*provisioning.Instance, error) {
	inst, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	result := &provisioning.Instance{
		ID:        inst.ID,
		UserID:    inst.UserID,
		Status:    inst.Status,
		SubDomain: inst.Subdomain,
		Namespace: inst.Namespace,
		CreatedAt: time.Time{},
	}

	if inst.CreatedAt.Valid {
		result.CreatedAt = inst.CreatedAt.Time
	}

	if inst.DeployedAt.Valid {
		result.DeployedAt = &inst.DeployedAt.Time
	}

	return result, nil
}
