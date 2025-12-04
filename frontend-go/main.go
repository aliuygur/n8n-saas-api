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
	authHandlers := handlers.NewAuthHandlers()
	dashboardHandlers := handlers.NewDashboardHandlers(apiClient)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	staticFS := http.FS(content)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

	// Routes
	r.Get("/", handlers.HandleHome())
	r.Get("/login", authHandlers.HandleLogin)
	r.Get("/register", authHandlers.HandleRegister)
	r.Get("/dashboard", authHandlers.RequireAuth(dashboardHandlers.HandleDashboard(authHandlers.GetCurrentUser)))

	// Auth endpoints
	r.Post("/auth/login", authHandlers.HandleLoginPost)
	r.Post("/auth/register", authHandlers.HandleRegisterPost)
	r.Post("/logout", authHandlers.HandleLogout)

	// HTMX endpoints (protected)
	r.Post("/api/instances", authHandlers.RequireAuth(dashboardHandlers.HandleCreateInstance))
	r.Get("/api/instances", authHandlers.RequireAuth(dashboardHandlers.HandleListInstances))
	r.Delete("/api/instances/{id}", authHandlers.RequireAuth(dashboardHandlers.HandleDeleteInstance))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
