package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

// HandleGoogleLogin redirects to Google OAuth
//
//encore:api public raw method=GET path=/auth/google
func (s *Service) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate a random state token for CSRF protection
	state := lo.RandomString(32, lo.LettersCharset)

	// TODO: Store state in cache/session for validation in callback
	// For now, we'll just generate the URL
	authURL := s.oauth2Config.AuthCodeURL(state)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GoogleUserInfo represents the user info returned from Google
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// HandleGoogleCallback handles the OAuth callback from Google
//
//encore:api public raw method=GET path=/auth/google/callback
func (s *Service) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	if code == "" {
		// Redirect to frontend with error
		http.Redirect(w, r, "http://localhost:5173/login?error=no_code", http.StatusSeeOther)
		return
	}

	ctx := r.Context()

	// TODO: Validate state token to prevent CSRF

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		rlog.Error("Failed to exchange code", "error", err)
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Get user info from Google
	client := s.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		rlog.Error("Failed to get user info", "error", err)
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		rlog.Error("Failed to get user info", "status", resp.StatusCode, "body", string(body))
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		rlog.Error("Failed to decode user info", "error", err)
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Create or get user in database
	queries := db.New(s.db)

	var user db.User
	existingUser, err := queries.GetUserByEmail(ctx, googleUser.Email)
	if err == sql.ErrNoRows {
		// Create new user
		user, err = queries.CreateUser(ctx, db.CreateUserParams{
			Email: googleUser.Email,
			Name:  googleUser.Name,
			Picture: sql.NullString{
				String: googleUser.Picture,
				Valid:  googleUser.Picture != "",
			},
		})
		if err != nil {
			rlog.Error("Failed to create user", "error", err)
			http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
			return
		}
		rlog.Info("New user created", "user_id", user.ID, "email", user.Email)
	} else if err != nil {
		rlog.Error("Failed to get user", "error", err)
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	} else {
		user = existingUser
	}

	// Update last login
	user, err = queries.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to update last login: %v", err)
		// Don't fail the login, just log the error
	}

	// Create JWT token
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "instol.cloud",
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		rlog.Error("Failed to create JWT token", "error", err)
		http.Redirect(w, r, "http://localhost:5173/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	rlog.Info("JWT token created", "user_id", user.ID)

	// Redirect to frontend with JWT token in URL
	// Frontend will extract the token and store it in localStorage
	http.Redirect(w, r, fmt.Sprintf("http://localhost:5173/auth/callback?token=%s", tokenString), http.StatusSeeOther)
}
