package provisioning

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
