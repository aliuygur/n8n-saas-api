package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/aliuygur/n8n-saas-api/frontend-go/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed static
var content embed.FS

func main() {
	// Initialize handlers
	apiClient := NewAPIClient()
	authHandlers := handlers.NewAuthHandlers(apiClient)
	dashboardHandlers := handlers.NewDashboardHandlers(apiClient, authHandlers)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	staticFS := http.FS(content)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

	// Public routes
	r.Get("/", handlers.HandleHome())
	r.Get("/login", authHandlers.HandleLogin)
	r.Post("/logout", authHandlers.HandleLogout)

	// Google OAuth routes
	r.Get("/auth/google", authHandlers.HandleGoogleLogin)
	r.Get("/auth/google/callback", authHandlers.HandleGoogleCallback)

	// Protected routes group
	r.Group(func(r chi.Router) {
		r.Use(authHandlers.RequireAuth)

		// Dashboard pages
		r.Get("/dashboard", dashboardHandlers.HandleDashboard(authHandlers.GetCurrentUser))
		r.Get("/create-instance", dashboardHandlers.HandleCreateInstancePage(authHandlers.GetCurrentUser))

		// API endpoints
		r.Post("/api/instances", dashboardHandlers.HandleCreateInstance)
		r.Get("/api/instances", dashboardHandlers.HandleListInstances)
		r.Delete("/api/instances/{id}", dashboardHandlers.HandleDeleteInstance)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
