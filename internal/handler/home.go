package handler

import (
	"net/http"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	// Check if this is a subdomain request that should be proxied to n8n
	// This must happen BEFORE any route matching to avoid conflicts with /static/ routes
	host := r.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// If it's a subdomain (not www.ranx.cloud), proxy to n8n instance
	if host != "www.ranx.cloud" && host != "ranx.cloud" && strings.HasSuffix(host, ".ranx.cloud") {
		h.ProxyHandler(w, r)
		return
	}

	// Since this is registered as "GET /", it will catch all unmatched GET requests
	// Check if the path is exactly "/" for home page, otherwise show 404
	if r.URL.Path != "/" {
		h.NotFound(w, r)
		return
	}
	lo.Must0(components.HomePage().Render(r.Context(), w))
}

// NotFound renders the 404 page
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	lo.Must0(components.NotFoundPage().Render(r.Context(), w))
}
