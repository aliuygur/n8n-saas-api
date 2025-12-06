package api

import (
	"database/sql"

	"encore.dev/storage/sqldb"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Encore database definition
var apiDB = sqldb.NewDatabase("api", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

// Service handles API endpoints for the React Router frontend
//
//encore:service
type Service struct {
	db           *sql.DB
	oauth2Config *oauth2.Config
	jwtSecret    []byte
}

var svc *Service

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
		GoogleRedirectURL:  "http://localhost:8080/auth/google/callback",
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

	db := apiDB.Stdlib()

	s := &Service{
		db:           db,
		oauth2Config: oauth2Config,
		jwtSecret:    []byte(config.JWTSecret),
	}

	svc = s
	return s, nil
}
