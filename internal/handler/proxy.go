package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
)

// ProxyHandler proxies requests to n8n instances based on subdomain
// Extracts subdomain from Host header (e.g., subdomain.n8n.instol.cloud)
// Queries instances table to find namespace
// Forwards request to http://n8n-main.{namespace}.svc.cluster.local
func (h *Handler) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appctx.GetLogger(ctx)

	// Extract subdomain from Host header
	host := r.Host
	subdomain := resolveTenant(host)
	if subdomain == "" {
		l.Warn("Invalid host format", slog.String("host", host))
		http.Error(w, "Invalid host", http.StatusBadRequest)
		return
	}

	l.Debug("Resolving tenant", slog.String("host", host), slog.String("subdomain", subdomain))

	// Query database for instance
	instance, err := h.services.GetInstanceBySubdomain(ctx, subdomain)
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
	}

	proxy.ServeHTTP(w, r)
}

// resolveTenant extracts subdomain from host
// Example: ali.n8n.instol.cloud -> ali
func resolveTenant(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	parts := strings.Split(host, ".")
	return parts[0]
}
