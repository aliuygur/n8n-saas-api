package handlers

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/frontend-go/views/pages"
)

// HandleHome returns a handler for the home page
func HandleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pages.Home().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
