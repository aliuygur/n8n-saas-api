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
	ExpiresAt time.Time
}

// AuthHandlers handles authentication-related requests
type AuthHandlers struct {
	users    map[string]*views.User // email -> User
	sessions map[string]*Session    // sessionID -> Session
	mu       sync.RWMutex
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers() *AuthHandlers {
	h := &AuthHandlers{
		users:    make(map[string]*views.User),
		sessions: make(map[string]*Session),
	}

	// Add a demo user
	h.users["demo@instol.cloud"] = &views.User{
		ID:        "demo-user",
		Email:     "demo@instol.cloud",
		Password:  "demo123", // In production, use bcrypt
		FirstName: "Demo",
		LastName:  "User",
		CreatedAt: time.Now(),
	}

	return h
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

func (h *AuthHandlers) HandleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	h.mu.RLock()
	user, exists := h.users[email]
	h.mu.RUnlock()

	if !exists || user.Password != password {
		w.Header().Set("Content-Type", "text/html")
		if err := pages.LoginError("Invalid email or password").Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.sessions[sessionID] = &Session{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	h.mu.Unlock()

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400, // 24 hours
	})

	w.Header().Set("Content-Type", "text/html")
	if err := pages.LoginSuccess().Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AuthHandlers) HandleRegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")

	h.mu.Lock()
	if _, exists := h.users[email]; exists {
		h.mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		if err := pages.RegisterError("Email already registered").Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Create user
	userID, _ := generateSessionID()
	user := &views.User{
		ID:        userID,
		Email:     email,
		Password:  password, // In production, use bcrypt.GenerateFromPassword
		FirstName: firstName,
		LastName:  lastName,
		CreatedAt: time.Now(),
	}
	h.users[email] = user
	h.mu.Unlock()

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.sessions[sessionID] = &Session{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	h.mu.Unlock()

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
	})

	log.Printf("New user registered: %s", email)

	w.Header().Set("Content-Type", "text/html")
	if err := pages.RegisterSuccess().Render(r.Context(), w); err != nil {
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

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, user := range h.users {
		if user.ID == session.UserID {
			return user
		}
	}

	return nil
}

func (h *AuthHandlers) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.GetCurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
