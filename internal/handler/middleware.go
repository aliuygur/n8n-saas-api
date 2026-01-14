package handler

import (
	"context"
	"net/http"
	"strings"
)

// HostRouter middleware routes requests based on the Host header
// *.ranx.cloud (except www and apex) -> proxy handler
// www.ranx.cloud, ranx.cloud -> mux routes
func (h *Handler) HostRouter(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// Remove port if present
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		// If it's a subdomain (not www or apex), proxy to n8n instance
		if host != "www.ranx.cloud" && host != "ranx.cloud" && strings.HasSuffix(host, ".ranx.cloud") {
			h.ProxyHandler(w, r)
			return
		}

		// Otherwise, use the normal mux
		mux.ServeHTTP(w, r)
	})
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userContextKey contextKey = "user"
)

// AuthMiddleware checks for valid authentication and adds user to context
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.GetUserFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware checks for authentication but doesn't require it
// Adds user to context if authenticated, otherwise continues without user
func (h *Handler) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.GetUserFromRequest(r)
		if err == nil {
			// Add user to context if authenticated
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Continue without user
		next.ServeHTTP(w, r)
	})
}

// MustGetUser retrieves the user from context, panics if not found
// Should only be used after AuthMiddleware
func MustGetUser(ctx context.Context) *JWTClaims {
	user, ok := ctx.Value(userContextKey).(*JWTClaims)
	if !ok {
		panic("user not found in context - ensure AuthMiddleware is applied")
	}
	return user
}

// GetUser retrieves the user from context, returns nil if not found
func GetUser(ctx context.Context) *JWTClaims {
	user, ok := ctx.Value(userContextKey).(*JWTClaims)
	if !ok {
		return nil
	}
	return user
}

// requireAuth is a helper to wrap handlers that require authentication
func (h *Handler) requireAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := h.GetUserFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		handlerFunc(w, r.WithContext(ctx))
	}
}

// requireAuthAPI is a helper for API endpoints that require authentication
// Returns 401 instead of redirect
func (h *Handler) requireAuthAPI(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := h.GetUserFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		handlerFunc(w, r.WithContext(ctx))
	}
}
