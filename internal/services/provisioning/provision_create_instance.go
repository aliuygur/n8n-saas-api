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
	UserID    string `json:"user_id"`
	Subdomain string `json:"subdomain"`
}

type CreateInstanceResponse struct {
	InstanceID int    `json:"instance_id"`
	Status     string `json:"status"`
	Domain     string `json:"domain"`
}

//encore:api private
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	// Validate required fields
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Subdomain == "" {
		return nil, fmt.Errorf("subdomain is required")
	}

	// Validate subdomain
	if err := validateSubdomain(req.Subdomain); err != nil {
		return nil, fmt.Errorf("invalid subdomain: %w", err)
	}

	// Check if subdomain is already taken
	queries := db.New(s.db)
	subdomainExists, err := queries.CheckSubdomainExists(ctx, req.Subdomain)
	if err != nil {
		return nil, fmt.Errorf("failed to check subdomain availability: %w", err)
	}
	if subdomainExists {
		return nil, fmt.Errorf("subdomain '%s' is already taken", req.Subdomain)
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

	instance, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
		UserID:         req.UserID,
		GkeClusterName: s.config.DefaultClusterName,
		GkeProjectID:   s.config.DefaultProjectID,
		GkeZone:        s.config.DefaultZone,
		Namespace:      namespace,
		Subdomain:      req.Subdomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Start deployment
	domain := fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain)
	if err := s.deployInstance(ctx, instance, encryptionKey); err != nil {
		return nil, fmt.Errorf("failed to deploy instance: %w", err)
	}

	return &CreateInstanceResponse{
		InstanceID: int(instance.ID),
		Status:     instance.Status,
		Domain:     domain,
	}, nil
}

func (s *Service) deployInstance(ctx context.Context, instance db.Instance, encryptionKey string) error {
	queries := db.New(s.db)

	rlog.Info("Starting deployment", "instance_id", instance.ID)

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Deploy n8n instance
	domain := fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain)
	n8nInstance := gke.N8NInstance{
		Namespace:     instance.Namespace,
		CPURequest:    "150m",
		MemoryRequest: "512Mi",
		CPULimit:      "500m",
		MemoryLimit:   "1Gi",
		StorageSize:   "5Gi",
		EncryptionKey: encryptionKey,
		BaseURL:       domain,
	}

	if err := s.gke.DeployN8NInstance(ctx, n8nInstance); err != nil {
		return fmt.Errorf("failed to deploy n8n: %w", err)
	}

	// Mark as deployed
	_, err := queries.UpdateInstanceDeployed(ctx, db.UpdateInstanceDeployedParams{
		ID:     instance.ID,
		Status: "deployed",
	})
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Add Cloudflare tunnel route for external access
	serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", instance.Namespace)
	if err := s.cloudflare.AddTunnelRoute(ctx, domain, serviceURL); err != nil {
		rlog.Error("Failed to add Cloudflare tunnel route",
			"error", err,
			"domain", domain,
			"service_url", serviceURL)
		// Don't fail the deployment if tunnel route creation fails
	} else {
		rlog.Info("Successfully added Cloudflare tunnel route",
			"domain", domain,
			"service_url", serviceURL)
	}

	rlog.Info("Deployment completed successfully", "instance_id", instance.ID, "namespace", instance.Namespace)
	return nil
}
