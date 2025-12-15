package services

import (
	"database/sql"

	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning"
)

type Service struct {
	db         *sql.DB
	cloudflare *cloudflare.Client
	gke        *provisioning.Client
}

func NewService(db *sql.DB, config *config.Config) (*Service, error) {
	cfClient := cloudflare.NewClient(cloudflare.Config{
		APIToken:  config.Cloudflare.APIToken,
		TunnelID:  config.Cloudflare.TunnelID,
		AccountID: config.Cloudflare.AccountID,
		ZoneID:    config.Cloudflare.ZoneID,
	})

	gke, err := provisioning.NewClient()
	if err != nil {
		return nil, err
	}

	return &Service{db: db, cloudflare: cfClient, gke: gke}, nil
}
