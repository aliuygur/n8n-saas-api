package frontend

import (
	"net/http"

	"encore.dev"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Encore database definition
var apiDB = sqldb.NewDatabase("frontend", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

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

func initService() (*Service, error) {
	// TODO: Load these from Encore secrets
	config := Config{
		GoogleClientID:     "YOUR_GOOGLE_CLIENT_ID",
		GoogleClientSecret: "YOUR_GOOGLE_CLIENT_SECRET",
		GoogleRedirectURL:  encore.Meta().APIBaseURL.String() + "/auth/google/callback",
		JWTSecret:          "your-secret-key-change-this-in-production",
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	s := &Service{
		db:           db.New(apiDB.Stdlib()),
		oauth2Config: oauth2Config,
		jwtSecret:    []byte(config.JWTSecret),
	}

	return s, nil
}

// ServeStatic serves static files (CSS, images, etc.)
//
//encore:api public raw path=/static/*path
func (s *Service) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Serve from embedded static files
	fs := http.FileServer(http.Dir("./internal/services/frontend/static"))
	http.StripPrefix("/static/", fs).ServeHTTP(w, r)
}

// withUserLock executes a function while holding an advisory lock for the given user ID
func (s *Service) withUserLock(r *http.Request, userID string, fn func() error) error {
	lo.Must0(s.db.AcquireUserLock(r.Context(), userID))
	defer lo.Must0(s.db.ReleaseUserLock(r.Context(), userID))
	return fn()
}

// handleError renders error page
func (s *Service) handleError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	w.WriteHeader(statusCode)
	component := components.ErrorPage(message)
	if err := component.Render(r.Context(), w); err != nil {
		rlog.Error("failed to render error page", "error", err)
		http.Error(w, message, statusCode)
	}
}
