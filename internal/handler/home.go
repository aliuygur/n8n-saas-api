package handler

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Subdomain routing is now handled in NotFoundHandlerWrapper
	// This handler only processes requests for www.ranx.cloud and ranx.cloud
	lo.Must0(components.HomePage().Render(r.Context(), w))
}

// Home renders the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.HomePage().Render(r.Context(), w))
}

// NotFound renders the 404 page
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	lo.Must0(components.NotFoundPage().Render(r.Context(), w))
}
