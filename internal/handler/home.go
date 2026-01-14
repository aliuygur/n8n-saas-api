package handler

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
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
