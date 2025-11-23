// Package provisioning handles N8N instance creation with cost optimization features.
//
// Cost Optimization Features:
// - Spot instances: Enabled by default for 60-91% cost savings on compute
// - Efficient resource allocation: 150m CPU, 512Mi memory requests
// - Optimized storage: 5Gi Premium SSD for balance of cost and performance
//
// Monthly cost per instance with spot pricing (us-central1):
// - CPU (0.15 vCPU): ~$1.73 (vs $4.86 regular)
// - Memory (0.5 GiB): ~$0.79 (vs $2.23 regular)
// - Storage (5GB): ~$0.85
// - Total: ~$3.37/month (vs ~$7.94 without spot)
package provisioning

import (
	"context"
	"fmt"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/gke"
)

// Create Instance API types
type CreateInstanceRequest struct {
	UserID string `json:"user_id"`
	Domain string `json:"domain"`
}

type CreateInstanceResponse struct {
	InstanceID   int    `json:"instance_id"`
	DeploymentID int    `json:"deployment_id"`
	Status       string `json:"status"`
}

//encore:api private
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	// Validate required fields
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	// Generate unique namespace
	namespace, err := s.generateUniqueNamespace(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique namespace: %w", err)
	}

	// Generate secure credentials
	encryptionKey, err := generateSecureKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Create database record
	queries := db.New(s.db)

	instance, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
		UserID:         req.UserID,
		GkeClusterName: s.config.DefaultClusterName,
		GkeProjectID:   s.config.DefaultProjectID,
		GkeZone:        s.config.DefaultZone,
		Namespace:      namespace,
		Domain:         fmt.Sprintf("https://%s.instol.cloud", namespace),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Create deployment record
	deployment, err := queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		InstanceID: instance.ID,
		Operation:  "deploy",
		Details:    []byte(`{"encryption_key":"***"}`),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Start async deployment
	go s.deployInstanceAsync(context.Background(), instance, deployment.ID, encryptionKey, "")

	return &CreateInstanceResponse{
		InstanceID:   int(instance.ID),
		DeploymentID: int(deployment.ID),
		Status:       instance.Status,
	}, nil
}

func (s *Service) deployInstanceAsync(ctx context.Context, instance db.Instance, deploymentID int32, encryptionKey, dbPassword string) {
	queries := db.New(s.db)

	rlog.Info("Starting deployment", "instance_id", instance.ID, "deployment_id", deploymentID)

	// Update deployment status to running
	_, err := queries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID:           deploymentID,
		Status:       "running",
		ErrorMessage: "",
	})
	if err != nil {
		rlog.Error("Failed to update deployment status", "error", err)
		return
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to connect to cluster: %v", err))
		return
	}

	// Deploy n8n instance with spot instances for cost savings (60-91% discount)
	n8nInstance := gke.N8NInstance{
		Namespace:     instance.Namespace,
		CPURequest:    "150m",
		MemoryRequest: "512Mi",
		CPULimit:      "500m",
		MemoryLimit:   "1Gi",
		StorageSize:   "5Gi",
		EncryptionKey: encryptionKey,
		UseSpotNodes:  s.config.UseSpotInstances, // Configurable spot instance usage
		BaseURL:       instance.Domain,
	}

	if err := s.gke.DeployN8NInstance(ctx, n8nInstance); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to deploy n8n: %v", err))
		return
	}

	// Mark as deployed
	_, err = queries.UpdateInstanceDeployed(ctx, db.UpdateInstanceDeployedParams{
		ID:     instance.ID,
		Status: "deployed",
	})
	if err != nil {
		rlog.Error("Failed to update instance status", "error", err)
	}

	_, err = queries.UpdateDeploymentCompleted(ctx, deploymentID)
	if err != nil {
		rlog.Error("Failed to mark deployment completed", "error", err)
	}

	// Add Cloudflare tunnel route for external access
	serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", instance.Namespace)
	if err := s.cloudflare.AddTunnelRoute(ctx, instance.Domain, serviceURL); err != nil {
		rlog.Error("Failed to add Cloudflare tunnel route",
			"error", err,
			"domain", instance.Domain,
			"service_url", serviceURL)
		// Don't fail the deployment if tunnel route creation fails
	} else {
		rlog.Info("Successfully added Cloudflare tunnel route",
			"domain", instance.Domain,
			"service_url", serviceURL)
	}

	rlog.Info("Deployment completed successfully", "instance_id", instance.ID, "namespace", instance.Namespace)
}
