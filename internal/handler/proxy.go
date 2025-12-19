package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/services"
)

// ProxyHandler proxies requests to n8n instances based on subdomain
// Extracts subdomain from Host header (e.g., subdomain.n8n.instol.cloud)
// Queries instances table to find namespace
// Forwards request to http://n8n-main.{namespace}.svc.cluster.local
func (h *Handler) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appctx.GetLogger(ctx)

	// Extract subdomain and get instance (with caching)
	instance, subdomain, err := h.resolveTenant(ctx, r.Host)
	if err != nil {
		if ok := apperrs.CodeIs(err, apperrs.CodeNotFound); ok {
			l.Warn("Instance not found", slog.String("subdomain", subdomain))
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		l.Error("Failed to get instance", slog.String("subdomain", subdomain), slog.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build target host
	targetHost := fmt.Sprintf("n8n-main.%s.svc.cluster.local", instance.Namespace)

	l.Info("Proxying request",
		slog.String("subdomain", subdomain),
		slog.String("namespace", instance.Namespace),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery))

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = targetHost
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", "https")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			l.Error("Proxy error",
				slog.String("subdomain", subdomain),
				slog.String("target_host", targetHost),
				slog.Any("error", err))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
		FlushInterval: -1, // WebSocket support
	}

	proxy.ServeHTTP(w, r)
}

// resolveTenant extracts subdomain from host and retrieves the instance
// Example: ali.n8n.instol.cloud -> ali
// Uses in-memory cache with TTL to reduce database queries
func (h *Handler) resolveTenant(ctx context.Context, host string) (*services.Instance, string, error) {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	parts := strings.Split(host, ".")
	subdomain := parts[0]

	if subdomain == "" {
		return nil, "", apperrs.Client(apperrs.CodeInvalidInput, "invalid host format")
	}

	// Check cache first
	if cached, ok := h.instanceCache.Load(subdomain); ok {
		if entry, ok := cached.(*instanceCacheEntry); ok {
			// Check if cache entry is still valid
			if time.Now().Before(entry.expiresAt) {
				return entry.instance, subdomain, nil
			}
			// Cache expired, remove it
			h.instanceCache.Delete(subdomain)
		}
	}

	// Cache miss or expired - fetch from database
	instance, err := h.services.GetInstanceBySubdomain(ctx, subdomain)
	if err != nil {
		return nil, subdomain, err
	}

	// Store in cache with expiration
	entry := &instanceCacheEntry{
		instance:  instance,
		expiresAt: time.Now().Add(instanceCacheTTL),
	}
	h.instanceCache.Store(subdomain, entry)

	return instance, subdomain, nil
}
