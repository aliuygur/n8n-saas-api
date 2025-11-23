package provisioning

import (
	"context"
)

type ResetTunnelConfigResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ResetTunnelConfig resets the Cloudflare tunnel configuration to a clean state
//
//encore:api public method=POST path=/admin/reset-tunnel-config
func (s *Service) ResetTunnelConfig(ctx context.Context) (*ResetTunnelConfigResponse, error) {
	if err := s.cloudflare.ResetTunnelConfig(ctx); err != nil {
		return &ResetTunnelConfigResponse{
			Success: false,
			Message: "Failed to reset tunnel config: " + err.Error(),
		}, err
	}

	return &ResetTunnelConfigResponse{
		Success: true,
		Message: "Tunnel configuration has been reset successfully",
	}, nil
}
