package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aliuygur/n8n-saas-api/frontend-go/views"
	"github.com/aliuygur/n8n-saas-api/frontend-go/views/pages"
)

// Session represents a user session
type Session struct {
	UserID    string
	APIToken  string // Backend API token
	ExpiresAt time.Time
}

// AuthHandlers handles authentication-related requests
type AuthHandlers struct {
	sessions  map[string]*Session // sessionID -> Session
	apiClient APIClient
	mu        sync.RWMutex
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(apiClient APIClient) *AuthHandlers {
	return &AuthHandlers{
		sessions:  make(map[string]*Session),
		apiClient: apiClient,
	}
}

func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := pages.Login().Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := pages.Register().Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		h.mu.Lock()
		delete(h.sessions, cookie.Value)
		h.mu.Unlock()
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleGoogleLogin redirects to Google OAuth
func (h *AuthHandlers) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	authURL, err := h.apiClient.GetGoogleLoginURL()
	if err != nil {
		log.Printf("Failed to get Google login URL: %v", err)
		http.Error(w, "Failed to initiate Google login", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback handles the OAuth callback from Google
func (h *AuthHandlers) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Redirect(w, r, "/login?error=no_code", http.StatusSeeOther)
		return
	}

	// Exchange code for session token from backend API
	result, err := h.apiClient.HandleGoogleCallback(code, state)
	if err != nil {
		log.Printf("Google callback failed: %v", err)
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	// Create session ID for browser cookie
	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store API token in server-side session
	h.mu.Lock()
	h.sessions[sessionID] = &Session{
		UserID:    result.User.Email,
		APIToken:  result.SessionToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}
	h.mu.Unlock()

	// Set session cookie for browser
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,            // Set to true in production with HTTPS
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandlers) GetCurrentUser(r *http.Request) *views.User {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	h.mu.RLock()
	session, exists := h.sessions[cookie.Value]
	h.mu.RUnlock()

	if !exists || session.ExpiresAt.Before(time.Now()) {
		return nil
	}

	// Return a basic user object with the session's user ID (email)
	// In a real app, you might fetch full user details from a database
	return &views.User{
		ID:    session.UserID,
		Email: session.UserID,
	}
}

func (h *AuthHandlers) GetAPIToken(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	session, exists := h.sessions[cookie.Value]
	if !exists || session.ExpiresAt.Before(time.Now()) {
		return ""
	}

	return session.APIToken
}

// RequireAuth is a middleware that protects routes requiring authentication
func (h *AuthHandlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := h.GetCurrentUser(r)
		if user == nil {
			// For HTMX requests, return 401 instead of redirect
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
