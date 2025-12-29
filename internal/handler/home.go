package handler

import (
	"net/http"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Host != "www.ranx.cloud" && strings.HasSuffix(r.Host, ".ranx.cloud") {
		h.ProxyHandler(w, r)
		return
	}

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
