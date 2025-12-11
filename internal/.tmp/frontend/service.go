package frontend

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"golang.org/x/oauth2"
)

// Service handles API endpoints for the React Router frontend
//
//encore:service
type Service struct {
	db           *db.Queries
	oauth2Config *oauth2.Config
	jwtSecret    []byte
}

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
}

// ServeStatic serves static files (CSS, images, etc.)
//
//encore:api public raw path=/static/*path
func (s *Service) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Serve from embedded static files
	fs := http.FileServer(http.Dir("./internal/services/frontend/static"))
	http.StripPrefix("/static/", fs).ServeHTTP(w, r)
}
