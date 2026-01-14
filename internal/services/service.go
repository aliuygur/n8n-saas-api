package services

import (
	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning"
	"github.com/aliuygur/n8n-saas-api/pkg/lemonsqueezy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool         *pgxpool.Pool
	cloudflare   *cloudflare.Client
	gke          *provisioning.Client
	lemonsqueezy *lemonsqueezy.Client
	config       *config.Config
}

func NewService(pool *pgxpool.Pool, config *config.Config) (*Service, error) {
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

	lsClient := lemonsqueezy.NewClient(lemonsqueezy.Config{
		APIKey:        config.LemonSqueezy.APIKey,
		WebhookSecret: config.LemonSqueezy.WebhookSecret,
	})

	return &Service{
		pool:         pool,
		cloudflare:   cfClient,
		gke:          gke,
		lemonsqueezy: lsClient,
		config:       config,
	}, nil
}
