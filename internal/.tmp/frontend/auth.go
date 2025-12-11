package frontend

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

// Login renders the login page
//
//encore:api public raw method=GET path=/login
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated via cookie
	cookie, err := r.Cookie("jwt")
	if err == nil && cookie.Value != "" {
		// User is authenticated, redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	lo.Must0(components.LoginPage().Render(r.Context(), w))
}

// Logout logs out the user by clearing the JWT cookie
//
//encore:api public raw method=GET path=/api/auth/logout
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the JWT cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleGoogleLogin redirects to Google OAuth
//
//encore:api public raw method=GET path=/auth/google
func (s *Service) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	rlog.Info("Initiating Google OAuth login")
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
		http.Redirect(w, r, "/login?error=no_code", http.StatusSeeOther)
		return
	}

	ctx := r.Context()

	// TODO: Validate state token to prevent CSRF

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		rlog.Error("Failed to exchange code", "error", err)
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Get user info from Google
	client := s.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		rlog.Error("Failed to get user info", "error", err)
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		rlog.Error("Failed to get user info", "status", resp.StatusCode, "body", string(body))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		rlog.Error("Failed to decode user info", "error", err)
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Create or get user in database

	var user db.User
	existingUser, err := s.db.GetUserByEmail(ctx, googleUser.Email)
	if err == sql.ErrNoRows {
		// Create new user
		user, err = s.db.CreateUser(ctx, db.CreateUserParams{
			Email: googleUser.Email,
			Name:  googleUser.Name,
		})
		if err != nil {
			rlog.Error("Failed to create user", "error", err)
			http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
			return
		}
		rlog.Info("New user created", "user_id", user.ID, "email", user.Email)
	} else if err != nil {
		rlog.Error("Failed to get user", "error", err)
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	} else {
		user = existingUser
	}

	// Update last login
	user, err = s.db.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		rlog.Error("Failed to update last login", "error", err)
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
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	rlog.Info("JWT token created", "user_id", user.ID)

	// Set JWT token in HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
