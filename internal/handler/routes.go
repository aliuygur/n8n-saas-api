package handler

import (
	"net/http"
)

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./internal/services/frontend/static"))))

	// Public routes (no auth required)
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /login", h.Login)
	mux.HandleFunc("GET /auth/google", h.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)

	// Auth required - Frontend pages (redirects to login)
	mux.HandleFunc("GET /dashboard", h.requireAuth(h.Dashboard))
	mux.HandleFunc("GET /create-instance", h.requireAuth(h.CreateInstancePage))
	mux.HandleFunc("GET /provisioning", h.requireAuth(h.ProvisioningPage))

	// Auth required - API endpoints (returns 401)
	mux.HandleFunc("GET /api/auth/logout", h.requireAuthAPI(h.Logout))
	mux.HandleFunc("GET /api/auth/me", h.requireAuthAPI(h.GetAuthMe))
	mux.HandleFunc("POST /api/create-instance", h.requireAuthAPI(h.CreateInstance))
	mux.HandleFunc("POST /api/check-subdomain", h.requireAuthAPI(h.CheckSubdomain))
	mux.HandleFunc("DELETE /instances/{id}", h.requireAuthAPI(h.DeleteInstance))
	mux.HandleFunc("GET /api/delete-modal/{id}", h.requireAuthAPI(h.DeleteModal))
	mux.HandleFunc("GET /api/provisioning-status", h.requireAuthAPI(h.GetProvisioningStatus))

	// Public webhooks (no auth)
	mux.HandleFunc("POST /api/webhooks/polar", h.PolarWebhook)
}
