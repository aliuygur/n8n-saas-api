package handler

import (
	"database/sql"
	"log/slog"

	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning"
	polargo "github.com/polarsource/polar-go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Handler holds all dependencies for HTTP handlers
type Handler struct {
	db                 *db.Queries
	provisioning       *provisioning.Client
	cloudflare         *cloudflare.Client
	polarClient        *polargo.Polar
	oauth2Config       *oauth2.Config
	jwtSecret          []byte
	config             *config.Config
	logger             *slog.Logger
	polarWebhookSecret string
}

// New creates a new Handler instance
func New(cfg *config.Config, database *sql.DB, logger *slog.Logger) (*Handler, error) {
	// Initialize database queries
	queries := db.New(database)

	// Initialize provisioning client
	// Tries in-cluster config first, then falls back to kubeconfig
	provisioningClient, err := provisioning.NewClient()
	if err != nil {
		return nil, err
	}

	// Initialize Cloudflare client
	cloudflareConfig := cloudflare.Config{
		APIToken:  cfg.Cloudflare.APIToken,
		TunnelID:  cfg.Cloudflare.TunnelID,
		AccountID: cfg.Cloudflare.AccountID,
		ZoneID:    cfg.Cloudflare.ZoneID,
	}
	cloudflareClient := cloudflare.NewClient(cloudflareConfig)

	// Initialize Polar client
	polarClient := polargo.New(
		polargo.WithServer(cfg.Polar.Server),
		polargo.WithSecurity(cfg.Polar.AccessToken),
	)

	// Initialize OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Google.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &Handler{
		db:                 queries,
		provisioning:       provisioningClient,
		cloudflare:         cloudflareClient,
		polarClient:        polarClient,
		oauth2Config:       oauth2Config,
		jwtSecret:          []byte(cfg.JWT.Secret),
		config:             cfg,
		logger:             logger,
		polarWebhookSecret: cfg.Polar.WebhookSecret,
	}, nil
}
