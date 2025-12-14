package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"github.com/aliuygur/n8n-saas-api/internal/types"
	"github.com/samber/lo"
)

// Instance represents an instance for internal use
type Instance struct {
	ID         string
	UserID     string
	Status     string
	Namespace  string
	SubDomain  string
	Subdomain  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeployedAt sql.NullTime
	DeletedAt  sql.NullTime
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

// generateUniqueNamespace creates a unique namespace by checking database
func (h *Handler) generateUniqueNamespace(ctx context.Context) (string, error) {

	// Try to find a unique namespace
	maxAttempts := 10
	for range maxAttempts {
		// Generate 8-character alphanumeric random string
		randomStr := lo.RandomString(8, append(lo.LowerCaseLettersCharset, lo.NumbersCharset...))

		// Format: n8n-{8-alphanumeric}
		namespace := fmt.Sprintf("n8n-%s", strings.ToLower(randomStr))
		// Truncate if too long (Kubernetes limit is 63 characters)
		if len(namespace) > 63 {
			namespace = namespace[:63]
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

type DeployInstanceRequest struct {
	InstanceID string
}

// deployInstance deploys an instance to Kubernetes
func (h *Handler) deployInstanceInternal(ctx context.Context, req *DeployInstanceRequest) error {
	h.logger.Info("Starting deployment", slog.String("instance_id", req.InstanceID))

	instance, err := h.db.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	encryptionKey, err := generateSecureKey(32)
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	// Deploy n8n instance
	domain := fmt.Sprintf("https://%s.instol.cloud", instance.Subdomain)
	n8nInstance := &n8ntemplates.N8N_V1{
		Namespace:     instance.Namespace,
		EncryptionKey: encryptionKey,
		BaseURL:       domain,
	}

	if err := h.provisioning.Apply(ctx, n8nInstance); err != nil {
		return fmt.Errorf("failed to deploy n8n: %w", err)
	}

	// Mark as deployed
	if _, err := h.db.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{
		ID:     instance.ID,
		Status: types.InstanceStatusDeployed,
	}); err != nil {
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

// TestDeployInstance is a temporary endpoint to test instance deployment
func (h *Handler) TestDeployInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Test configuration
	namespace := "n8n-test-abc2"
	subdomain := "aliko"

	h.logger.Info("Starting test deployment",
		slog.String("namespace", namespace),
		slog.String("subdomain", subdomain))

	// Generate encryption key
	encryptionKey, err := generateSecureKey(32)
	if err != nil {
		h.logger.Error("Failed to generate encryption key", slog.Any("error", err))
		http.Error(w, fmt.Sprintf("Failed to generate encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Deploy n8n instance directly without database interaction
	domain := fmt.Sprintf("https://%s.instol.cloud", subdomain)
	n8nInstance := &n8ntemplates.N8N_V1{
		Namespace:     namespace,
		EncryptionKey: encryptionKey,
		BaseURL:       domain,
	}

	if err := h.provisioning.Apply(ctx, n8nInstance); err != nil {
		h.logger.Error("Failed to deploy n8n", slog.Any("error", err))
		http.Error(w, fmt.Sprintf("Failed to deploy n8n: %v", err), http.StatusInternalServerError)
		return
	}

	// Add Cloudflare tunnel route for external access
	serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", namespace)
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

	h.logger.Info("Test deployment completed successfully",
		slog.String("namespace", namespace),
		slog.String("subdomain", subdomain))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"success","namespace":"%s","subdomain":"%s","domain":"%s"}`,
		namespace, subdomain, domain)
}

// CreateInstanceRequest represents a request to create an instance
type CreateInstanceRequest struct {
	InstanceID string
	UserID     string
	Subdomain  string
	// DeployNow field removed
}
