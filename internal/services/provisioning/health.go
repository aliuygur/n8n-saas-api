package provisioning

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// Health API endpoints

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

//encore:api private
func (s *Service) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Service: "n8n-provisioning-service",
	}, nil
}

// Helper functions
func (s *Service) markDeploymentFailed(ctx context.Context, queries *db.Queries, deploymentID int32, errorMsg string) {
	log.Printf("Deployment failed: deployment_id=%d error=%s", deploymentID, errorMsg)

	_, err := queries.UpdateDeploymentFailed(ctx, db.UpdateDeploymentFailedParams{
		ID:           deploymentID,
		ErrorMessage: errorMsg,
	})
	if err != nil {
		log.Printf("Failed to mark deployment as failed: %v", err)
	}
}

func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
