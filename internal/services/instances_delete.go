package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

type DeleteInstanceParams struct {
	UserID     string
	InstanceID string
}

// DeleteAllUserInstances deletes all instances for a given user.
// This is called when a subscription expires to clean up all user resources.
func (s *Service) DeleteAllUserInstances(ctx context.Context, userID string) error {
	log := appctx.GetLogger(ctx)
	queries := s.getDB()

	instances, err := queries.ListInstancesByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list user instances: %w", err)
	}

	if len(instances) == 0 {
		log.Info("no instances to delete for user", "user_id", userID)
		return nil
	}

	var deleteErrors []error
	for _, inst := range instances {
		if err := s.DeleteInstance(ctx, DeleteInstanceParams{
			UserID:     userID,
			InstanceID: inst.ID,
		}); err != nil {
			log.Error("failed to delete instance", "instance_id", inst.ID, "error", err)
			deleteErrors = append(deleteErrors, err)
		} else {
			log.Info("deleted instance", "instance_id", inst.ID)
		}
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("failed to delete %d of %d instances", len(deleteErrors), len(instances))
	}
	return nil
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

	// Delete PostgreSQL database and user for the n8n instance
	dbName := strings.ReplaceAll(instance.Namespace, "-", "_")
	if err := s.deleteInstanceDatabase(ctx, dbName); err != nil {
		// Log error but don't fail the deletion - the namespace is already gone
		l.Error("failed to delete instance database", "db_name", dbName, "error", err)
	} else {
		l.Debug("deleted instance database", "db_name", dbName)
	}

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
