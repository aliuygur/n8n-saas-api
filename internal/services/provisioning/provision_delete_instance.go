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

	// Create deletion deployment record
	deployment, err := queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		InstanceID: instance.ID,
		Operation:  "delete",
		Details:    []byte(`{}`),
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Start async deletion
	go s.deleteInstanceAsync(context.Background(), instance, deployment.ID)

	return nil
}

func (s *Service) deleteInstanceAsync(ctx context.Context, instance db.Instance, deploymentID int32) {
	queries := db.New(s.db)

	rlog.Info("Starting deletion", "instance_id", instance.ID)

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to connect to cluster: %v", err))
		return
	}

	// Delete n8n instance from GKE
	if err := s.gke.DeleteN8NInstance(ctx, instance.Namespace); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to delete n8n instance from GKE: %v", err))
		return
	}

	rlog.Info("GKE resources deleted successfully", "namespace", instance.Namespace)

	// Remove Cloudflare tunnel route
	if err := s.cloudflare.RemoveTunnelRoute(ctx, instance.Domain); err != nil {
		rlog.Error("Failed to remove Cloudflare tunnel route",
			"error", err,
			"domain", instance.Domain)
		// Don't fail the deletion if tunnel route removal fails
	} else {
		rlog.Info("Successfully removed Cloudflare tunnel route",
			"domain", instance.Domain)
	}

	// Soft delete from database
	_, err := queries.SoftDeleteInstance(ctx, instance.ID)
	if err != nil {
		rlog.Error("Failed to soft delete instance", "error", err)
	}

	// Mark deployment as completed
	_, err = queries.UpdateDeploymentCompleted(ctx, deploymentID)
	if err != nil {
		rlog.Error("Failed to mark deployment completed", "error", err)
	}

	rlog.Info("Deletion completed successfully", "instance_id", instance.ID)
}
