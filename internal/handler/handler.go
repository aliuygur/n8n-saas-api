package handler

import (
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Handler holds all dependencies for HTTP handlers
type Handler struct {
	oauth2Config       *oauth2.Config
	jwtSecret          []byte
	config             *config.Config
	polarWebhookSecret string

	services *services.Service
}

// New creates a new Handler instance
func New(cfg *config.Config, svc *services.Service) (*Handler, error) {

	// Initialize OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Server.BaseURL("/auth/google/callback"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &Handler{
		oauth2Config: oauth2Config,
		jwtSecret:    []byte(cfg.JWT.Secret),
		config:       cfg,
		services:     svc,
	}, nil
}
