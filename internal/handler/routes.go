package handler

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {

	// Register static files route first to avoid pattern conflicts
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))

	// Public routes (no auth required)
	mux.HandleFunc("/", h.HomeHandler)
	mux.HandleFunc("GET /login", h.Login)
	mux.HandleFunc("GET /auth/google", h.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)

	// Auth required - Frontend pages (redirects to login)
	mux.HandleFunc("GET /dashboard", h.requireAuth(h.Dashboard))
	mux.HandleFunc("GET /create-instance", h.requireAuth(h.CreateInstancePage))
	mux.HandleFunc("GET /provision", h.requireAuth(h.ProvisioningPage))
	mux.HandleFunc("GET /instances/{id}", h.requireAuth(h.InstanceDetail))
	mux.HandleFunc("GET /account", h.requireAuth(h.Account))
	// Keep old subscription route for backwards compatibility, redirect to account
	mux.HandleFunc("GET /subscription", h.requireAuth(h.Account))

	// Auth required - API endpoints (returns 401)
	mux.HandleFunc("GET /api/auth/logout", h.requireAuthAPI(h.Logout))
	mux.HandleFunc("GET /api/auth/me", h.requireAuthAPI(h.GetAuthMe))
	mux.HandleFunc("POST /api/create-instance", h.requireAuthAPI(h.CreateInstance))
	mux.HandleFunc("POST /api/check-subdomain", h.requireAuthAPI(h.CheckSubdomain))
	mux.HandleFunc("GET /api/check-instance-status", h.requireAuthAPI(h.CheckInstanceStatus))
	mux.HandleFunc("DELETE /instances/{id}", h.requireAuthAPI(h.DeleteInstance))

	// Legal pages (no auth)
	mux.HandleFunc("GET /pricing", PricingHandler)
	mux.HandleFunc("GET /terms", TermsOfServiceHandler)
	mux.HandleFunc("GET /privacy", PrivacyPolicyHandler)
	mux.HandleFunc("GET /refund-policy", RefundPolicyHandler)

	// Blog routes (no auth)
	mux.HandleFunc("GET /blog", h.BlogIndex)
	mux.HandleFunc("GET /blog/{slug}", h.BlogPost)

	// SEO routes (no auth)
	mux.HandleFunc("GET /sitemap.xml", h.Sitemap)

	// Public webhooks (no auth)
	// mux.HandleFunc("POST /api/webhooks/polar", h.PolarWebhook)
}

// NotFoundHandlerWrapper wraps the mux to intercept 404 responses and render custom page
// Also handles subdomain routing to proxy n8n instances
func (h *Handler) NotFoundHandlerWrapper(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// In Go 1.22, GET / matches everything as a catch-all
		// We need to check if the path is exactly "/" for home, or if it's a known route
		// For any other path, check if it would return 404

		if r.URL.Path != "/" && r.Method == "GET" {
			// Check if this exact path has a registered handler
			_, pattern := mux.Handler(r)

			// If it falls back to "GET /", it means no specific route matched
			// (since GET / is our catch-all home route)
			if pattern == "GET /" {
				h.NotFound(w, r)
				return
			}
		}

		// Serve normally
		mux.ServeHTTP(w, r)
	})
}
