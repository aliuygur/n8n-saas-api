package services

import (
	"context"
	"fmt"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

type DeleteInstanceParams struct {
	UserID     string
	InstanceID string
}

func (s *Service) DeleteInstance(ctx context.Context, params DeleteInstanceParams) error {
	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_delete_%s", params.UserID))
	defer releaseLock()

	l := appctx.GetLogger(ctx)

	instance, err := queries.GetInstance(ctx, params.InstanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.UserID != params.UserID {
		return apperrs.Client(apperrs.CodeForbidden, "user does not own the instance")
	}

	if err := s.gke.DeleteNamespace(ctx, instance.Namespace); err != nil {
		return apperrs.Server("failed to delete namespace from Kubernetes", err)
	}
	l.Debug("deleted namespace from Kubernetes", "namespace", instance.Namespace)

	if err := queries.DeleteInstance(ctx, params.InstanceID); err != nil {
		return apperrs.Server("failed to delete instance from database", err)
	}
	l.Debug("deleted instance from database", "instance_id", params.InstanceID)

	// Sync subscription quantity with LemonSqueezy
	if err := s.SyncSubscriptionQuantity(ctx, params.UserID); err != nil {
		// Log error but don't fail the deletion
		l.Error("failed to sync subscription quantity", "user_id", params.UserID, "error", err)
	}

	return nil
}
