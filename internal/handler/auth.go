package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appreq"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GoogleUserInfo represents the user info returned from Google
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// Login renders the login page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
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
func (h *Handler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appreq.GetLogger(ctx)

	l.Info("Initiating Google OAuth login")
	// Generate a random state token for CSRF protection
	state := lo.RandomString(32, lo.LettersCharset)

	// TODO: Store state in cache/session for validation in callback
	authURL := h.oauth2Config.AuthCodeURL(state)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback handles the OAuth callback from Google
func (h *Handler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	if code == "" {
		http.Redirect(w, r, "/login?error=no_code", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	l := appreq.GetLogger(ctx)

	// TODO: Validate state token to prevent CSRF

	// Exchange code for token
	token, err := h.oauth2Config.Exchange(ctx, code)
	if err != nil {
		l.Error("Failed to exchange code", slog.Any("error", err))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Get user info from Google
	client := h.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		l.Error("Failed to get user info", slog.Any("error", err))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		l.Error("Failed to get user info", slog.Int("status", resp.StatusCode), slog.String("body", string(body)))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		l.Error("Failed to decode user info", slog.Any("error", err))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	user, err := h.services.GetOrCreateUser(ctx, services.CreateUserParams{
		Email: googleUser.Email,
		Name:  googleUser.Name,
	})
	if err != nil {
		l.Error("Failed to get or create user", slog.Any("error", err))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	l.Info("User logged in via Google OAuth",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email))

	// Update last login
	err = h.services.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		l.Error("Failed to update last login", slog.Any("error", err))
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
	tokenString, err := jwtToken.SignedString(h.jwtSecret)
	if err != nil {
		l.Error("Failed to create JWT token", slog.Any("error", err))
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	l.Info("JWT token created", slog.String("user_id", user.ID))

	// Set JWT token in HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// GetAuthMe returns the current user information
func (h *Handler) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user := MustGetUser(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id": user.UserID,
		"email":   user.Email,
	})
}

// GetUserFromRequest extracts and validates the JWT from the request
func (h *Handler) GetUserFromRequest(r *http.Request) (*JWTClaims, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}
