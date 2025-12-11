package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/gke"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/samber/lo"
)

// Instance represents an instance for internal use
type Instance struct {
	ID             string
	UserID         string
	Status         string
	GkeClusterName string
	GkeProjectID   string
	GkeZone        string
	Namespace      string
	SubDomain      string
	Subdomain      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeployedAt     sql.NullTime
	DeletedAt      sql.NullTime
}

// checkSubdomainExistsInternal checks if a subdomain already exists
func (h *Handler) checkSubdomainExistsInternal(ctx context.Context, subdomain string) (bool, error) {
	_, err := h.db.GetInstanceBySubdomain(ctx, subdomain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check subdomain: %w", err)
	}
	return true, nil
}

// listInstancesInternal lists all instances for a user
func (h *Handler) listInstancesInternal(ctx context.Context, userID string) ([]Instance, error) {
	dbInstances, err := h.db.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	instances := make([]Instance, len(dbInstances))
	for i, inst := range dbInstances {
		createdAt := time.Time{}
		if inst.CreatedAt.Valid {
			createdAt = inst.CreatedAt.Time
		}
		updatedAt := time.Time{}
		if inst.UpdatedAt.Valid {
			updatedAt = inst.UpdatedAt.Time
		}

		instances[i] = Instance{
			ID:             inst.ID,
			UserID:         inst.UserID,
			Status:         inst.Status,
			GkeClusterName: inst.GkeClusterName,
			GkeProjectID:   inst.GkeProjectID,
			GkeZone:        inst.GkeZone,
			Namespace:      inst.Namespace,
			SubDomain:      fmt.Sprintf("https://%s.instol.cloud", inst.Subdomain),
			Subdomain:      inst.Subdomain,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			DeployedAt:     inst.DeployedAt,
			DeletedAt:      inst.DeletedAt,
		}
	}

	return instances, nil
}

// getInstanceInternal gets an instance by ID
func (h *Handler) getInstanceInternal(ctx context.Context, instanceID string) (*Instance, error) {
	inst, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	createdAt := time.Time{}
	if inst.CreatedAt.Valid {
		createdAt = inst.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if inst.UpdatedAt.Valid {
		updatedAt = inst.UpdatedAt.Time
	}

	return &Instance{
		ID:             inst.ID,
		UserID:         inst.UserID,
		Status:         inst.Status,
		GkeClusterName: inst.GkeClusterName,
		GkeProjectID:   inst.GkeProjectID,
		GkeZone:        inst.GkeZone,
		Namespace:      inst.Namespace,
		SubDomain:      fmt.Sprintf("https://%s.instol.cloud", inst.Subdomain),
		Subdomain:      inst.Subdomain,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		DeployedAt:     inst.DeployedAt,
		DeletedAt:      inst.DeletedAt,
	}, nil
}

// createInstanceInternal creates a new instance
func (h *Handler) createInstanceInternal(ctx context.Context, req CreateInstanceRequest) (*Instance, error) {
	// Validate required fields
	if req.InstanceID == "" {
		return nil, fmt.Errorf("instance_id is required")
	}
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Subdomain == "" {
		return nil, fmt.Errorf("subdomain is required")
	}

	// Validate subdomain
	if err := domainutils.ValidateSubdomain(req.Subdomain); err != nil {
		return nil, fmt.Errorf("invalid subdomain: %w", err)
	}

	// Check if subdomain is already taken
	subdomainExists, err := h.db.CheckSubdomainExists(ctx, req.Subdomain)
	if err != nil {
		return nil, fmt.Errorf("failed to check subdomain availability: %w", err)
	}
	if subdomainExists {
		return nil, fmt.Errorf("subdomain '%s' is already taken", req.Subdomain)
	}

	// Generate unique namespace
	namespace, err := h.generateUniqueNamespace(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique namespace: %w", err)
	}

	// Create instance record in database
	instance, err := h.db.CreateInstance(ctx, db.CreateInstanceParams{
		ID:             req.InstanceID,
		UserID:         req.UserID,
		GkeClusterName: h.config.GCP.ClusterName,
		GkeProjectID:   h.config.GCP.ProjectID,
		GkeZone:        h.config.GCP.Zone,
		Namespace:      namespace,
		Subdomain:      req.Subdomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	createdAt := time.Time{}
	if instance.CreatedAt.Valid {
		createdAt = instance.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if instance.UpdatedAt.Valid {
		updatedAt = instance.UpdatedAt.Time
	}

	result := &Instance{
		ID:             instance.ID,
		UserID:         instance.UserID,
		Status:         instance.Status,
		GkeClusterName: instance.GkeClusterName,
		GkeProjectID:   instance.GkeProjectID,
		GkeZone:        instance.GkeZone,
		Namespace:      instance.Namespace,
		SubDomain:      fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain),
		Subdomain:      instance.Subdomain,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		DeployedAt:     instance.DeployedAt,
		DeletedAt:      instance.DeletedAt,
	}

	// If DeployNow is false, just return the pending instance
	if !req.DeployNow {
		h.logger.Info("Pending instance created",
			slog.String("instance_id", instance.ID),
			slog.String("subdomain", req.Subdomain))
		return result, nil
	}

	// Use provided encryption key or generate a new one
	encryptionKey := req.EncryptionKey
	if encryptionKey == "" {
		encryptionKey, err = generateSecureKey(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
	}

	// Deploy the instance
	if err := h.deployInstance(ctx, instance, encryptionKey); err != nil {
		return nil, fmt.Errorf("failed to deploy instance: %w", err)
	}

	return result, nil
}

// deleteInstanceInternal deletes an instance
func (h *Handler) deleteInstanceInternal(ctx context.Context, instanceID string) error {
	// Get instance details
	instance, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Delete from GKE
	if err := h.gke.ConnectToCluster(ctx, instance.GkeClusterName, instance.GkeZone); err != nil {
		h.logger.Error("Failed to connect to cluster", slog.Any("error", err))
		// Continue with database deletion even if GKE deletion fails
	} else {
		if err := h.gke.DeleteN8NInstance(ctx, instance.Namespace); err != nil {
			h.logger.Error("Failed to delete namespace", slog.Any("error", err))
			// Continue with database deletion even if GKE deletion fails
		}
	}

	// Delete Cloudflare DNS record
	if err := h.cloudflare.DeleteDNSRecord(ctx, instance.Subdomain); err != nil {
		h.logger.Error("Failed to delete Cloudflare DNS record", slog.Any("error", err))
		// Continue with database deletion even if Cloudflare deletion fails
	}

	// Soft delete from database
	if err := h.db.DeleteInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to delete instance from database: %w", err)
	}

	return nil
}

// generateUniqueNamespace creates a unique namespace by checking database
func (h *Handler) generateUniqueNamespace(ctx context.Context, userID string) (string, error) {
	// Sanitize userID to be kubernetes-compliant
	sanitized := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(strings.ToLower(userID), "")

	// Try to find a unique namespace
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Generate 8-character alphanumeric random string
		randomStr := lo.RandomString(8, append(lo.LowerCaseLettersCharset, lo.NumbersCharset...))

		// Format: n8n-{userid}-{8-alphanumeric}
		namespace := fmt.Sprintf("n8n-%s-%s", sanitized, strings.ToLower(randomStr))

		// Truncate if too long (Kubernetes limit is 63 characters)
		if len(namespace) > 63 {
			maxUserIDLength := 63 - 4 - 8 - 2 // 63 - "n8n-" - random - "-"
			if len(sanitized) > maxUserIDLength {
				sanitized = sanitized[:maxUserIDLength]
			}
			namespace = fmt.Sprintf("n8n-%s-%s", sanitized, strings.ToLower(randomStr))
		}

		// Check if namespace exists in database
		exists, err := h.db.CheckNamespaceExists(ctx, namespace)
		if err != nil {
			return "", fmt.Errorf("failed to check namespace existence: %w", err)
		}

		if !exists {
			return namespace, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique namespace after %d attempts", maxAttempts)
}

// deployInstance deploys an instance to GKE
func (h *Handler) deployInstance(ctx context.Context, instance db.Instance, encryptionKey string) error {
	h.logger.Info("Starting deployment", slog.String("instance_id", instance.ID))

	// Connect to GKE cluster
	if err := h.gke.ConnectToCluster(ctx, h.config.GCP.ClusterName, h.config.GCP.Zone); err != nil {
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

	if err := h.gke.DeployN8NInstance(ctx, n8nInstance); err != nil {
		return fmt.Errorf("failed to deploy n8n: %w", err)
	}

	// Mark as deployed
	_, err := h.db.UpdateInstanceDeployed(ctx, db.UpdateInstanceDeployedParams{
		ID:     instance.ID,
		Status: "deployed",
	})
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Add Cloudflare tunnel route for external access
	serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", instance.Namespace)
	if err := h.cloudflare.AddTunnelRoute(ctx, domain, serviceURL); err != nil {
		h.logger.Error("Failed to add Cloudflare tunnel route",
			slog.Any("error", err),
			slog.String("domain", domain),
			slog.String("service_url", serviceURL))
		// Don't fail the deployment if tunnel route creation fails
	} else {
		h.logger.Info("Successfully added Cloudflare tunnel route",
			slog.String("domain", domain),
			slog.String("service_url", serviceURL))
	}

	h.logger.Info("Deployment completed successfully",
		slog.String("instance_id", instance.ID),
		slog.String("namespace", instance.Namespace))
	return nil
}

// generateSecureKey generates a secure random key of the specified byte length
func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateInstanceRequest represents a request to create an instance
type CreateInstanceRequest struct {
	InstanceID    string
	UserID        string
	Subdomain     string
	DeployNow     bool
	EncryptionKey string
}
