package handler

import (
	"sync"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// instanceCacheTTL defines how long instances are cached
	instanceCacheTTL = 5 * time.Minute
	// cacheCleanupInterval defines how often to clean up expired cache entries
	cacheCleanupInterval = 10 * time.Minute
)

// instanceCacheEntry holds cached instance data with expiration
type instanceCacheEntry struct {
	instance  *services.Instance
	expiresAt time.Time
}

// Handler holds all dependencies for HTTP handlers
type Handler struct {
	oauth2Config       *oauth2.Config
	jwtSecret          []byte
	config             *config.Config
	polarWebhookSecret string

	services *services.Service

	// instanceCache caches subdomain -> Instance mapping
	instanceCache sync.Map
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

	h := &Handler{
		oauth2Config: oauth2Config,
		jwtSecret:    []byte(cfg.JWT.Secret),
		config:       cfg,
		services:     svc,
	}

	// Start background cache cleanup
	go h.cleanupExpiredCacheEntries()

	return h, nil
}

// cleanupExpiredCacheEntries runs periodically to remove expired cache entries
func (h *Handler) cleanupExpiredCacheEntries() {
	ticker := time.NewTicker(cacheCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		h.instanceCache.Range(func(key, value any) bool {
			if entry, ok := value.(*instanceCacheEntry); ok {
				if now.After(entry.expiresAt) {
					h.instanceCache.Delete(key)
				}
			}
			return true
		})
	}
}
