package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/aliuygur/n8n-saas-api/frontend-go/views"
	"github.com/aliuygur/n8n-saas-api/frontend-go/views/pages"
	"github.com/go-chi/chi/v5"
)

type DashboardHandlers struct {
	apiClient    APIClient
	authHandlers *AuthHandlers
}

func NewDashboardHandlers(apiClient APIClient, authHandlers *AuthHandlers) *DashboardHandlers {
	return &DashboardHandlers{
		apiClient:    apiClient,
		authHandlers: authHandlers,
	}
}

func (h *DashboardHandlers) HandleDashboard(getCurrentUser func(*http.Request) *views.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getCurrentUser(r)
		if err := pages.Dashboard(user).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (h *DashboardHandlers) HandleCreateInstancePage(getCurrentUser func(*http.Request) *views.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getCurrentUser(r)
		if err := pages.CreateInstance(user).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (h *DashboardHandlers) HandleCreateInstance(w http.ResponseWriter, r *http.Request) {
	token := h.authHandlers.GetAPIToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subdomain := r.FormValue("subdomain")

	resp, err := h.apiClient.CreateInstance(token, CreateInstanceRequest{
		Subdomain: subdomain,
	})
	if err != nil {
		log.Printf("Error creating instance: %v", err)
		w.Header().Set("Content-Type", "text/html")
		// Extract user-friendly error message
		errMsg := err.Error()
		if err := pages.InstanceError(errMsg).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Instance created successfully: %s", resp.Subdomain)

	// Redirect to dashboard with success message
	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

func (h *DashboardHandlers) HandleListInstances(w http.ResponseWriter, r *http.Request) {
	token := h.authHandlers.GetAPIToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.apiClient.ListInstances(token)
	if err != nil {
		log.Printf("Error listing instances: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(resp.Instances) == 0 {
		if err := pages.EmptyInstances().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Convert to pages.Instance type
	instances := make([]pages.Instance, len(resp.Instances))
	for i, inst := range resp.Instances {
		// Extract subdomain from domain (format: https://subdomain.instol.cloud)
		subdomain := ""
		if len(inst.Domain) > 8 {
			// Remove https:// and .instol.cloud to get subdomain
			domain := inst.Domain[8:]             // Remove "https://"
			if idx := len(domain) - 13; idx > 0 { // Remove ".instol.cloud"
				subdomain = domain[:idx]
			}
		}
		instances[i] = pages.Instance{
			ID:        inst.ID,
			Subdomain: subdomain,
			Namespace: inst.Namespace,
			Status:    inst.Status,
			Domain:    inst.Domain,
			CreatedAt: inst.CreatedAt,
		}
	}

	if err := pages.InstanceListComponent(instances).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) HandleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	token := h.authHandlers.GetAPIToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid instance ID", http.StatusBadRequest)
		return
	}

	if err := h.apiClient.DeleteInstance(token, id); err != nil {
		log.Printf("Error deleting instance: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
