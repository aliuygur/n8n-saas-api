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

//encore:service
type Service struct {
	db           *sql.DB
	oauth2Config *oauth2.Config
}

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// initService initializes the API service
func initService() (*Service, error) {
	// TODO: Load these from Encore secrets
	config := Config{
		GoogleClientID:     "YOUR_GOOGLE_CLIENT_ID",
		GoogleClientSecret: "YOUR_GOOGLE_CLIENT_SECRET",
		GoogleRedirectURL:  "http://localhost:8080/auth/google/callback",
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

	return &Service{
		db:           apiDB.Stdlib(),
		oauth2Config: oauth2Config,
	}, nil
}

// GoogleUserInfo represents the user info returned from Google
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}
