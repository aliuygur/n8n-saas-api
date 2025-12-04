package provisioning

import (
	"context"
	"fmt"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Delete Instance API types
type DeleteInstanceRequest struct {
	InstanceID int `json:"instance_id"`
}

//encore:api private
func (s *Service) DeleteInstance(ctx context.Context, req *DeleteInstanceRequest) error {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Delete instance
	if err := s.deleteInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

func (s *Service) deleteInstance(ctx context.Context, instance db.Instance) error {
	queries := db.New(s.db)

	rlog.Info("Starting deletion", "instance_id", instance.ID)

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Delete n8n instance from GKE
	if err := s.gke.DeleteN8NInstance(ctx, instance.Namespace); err != nil {
		return fmt.Errorf("failed to delete n8n instance from GKE: %w", err)
	}

	rlog.Info("GKE resources deleted successfully", "namespace", instance.Namespace)

	// Remove Cloudflare tunnel route
	hostname := fmt.Sprintf("%s.instol.cloud", instance.Subdomain)
	if err := s.cloudflare.RemoveTunnelRoute(ctx, hostname); err != nil {
		rlog.Error("Failed to remove Cloudflare tunnel route",
			"error", err,
			"hostname", hostname)
		// Don't fail the deletion if tunnel route removal fails
	} else {
		rlog.Info("Successfully removed Cloudflare tunnel route",
			"hostname", hostname)
	}

	// Soft delete from database
	_, err := queries.SoftDeleteInstance(ctx, instance.ID)
	if err != nil {
		return fmt.Errorf("failed to soft delete instance: %w", err)
	}

	rlog.Info("Deletion completed successfully", "instance_id", instance.ID)
	return nil
}
