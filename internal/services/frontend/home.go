package frontend

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/samber/lo"
)

// Home renders the home page
//
//encore:api public raw method=GET path=/!fallback
func (s *Service) Home(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.HomePage().Render(r.Context(), w))
}
