package handler

import (
	"context"
	"fmt"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
)

// getInstanceForComponent gets an instance and converts it to provisioning.Instance for use with components
func (h *Handler) getInstanceForComponent(ctx context.Context, instanceID string) (*components.Instance, error) {
	inst, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	result := &components.Instance{
		ID:          inst.ID,
		InstanceURL: "",
		Status:      inst.Status,
	}

	return result, nil
}
