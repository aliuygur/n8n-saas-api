package handler

import (
	"embed"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {

	// Register static files route first to avoid pattern conflicts
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))

	// Public routes (no auth required)
	mux.HandleFunc("GET /", h.Home)
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
