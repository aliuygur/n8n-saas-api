package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aliuygur/n8n-saas-api/frontend-go/views"
	"github.com/aliuygur/n8n-saas-api/frontend-go/views/pages"
	"github.com/go-chi/chi/v5"
)

type DashboardHandlers struct {
	apiClient APIClient
}

func NewDashboardHandlers(apiClient APIClient) *DashboardHandlers {
	return &DashboardHandlers{
		apiClient: apiClient,
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

func (h *DashboardHandlers) HandleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subdomain := r.FormValue("subdomain")
	userID := r.FormValue("user_id")
	if userID == "" {
		userID = "demo-user"
	}

	resp, err := h.apiClient.CreateInstance(CreateInstanceRequest{
		UserID:    userID,
		Subdomain: subdomain,
	})
	if err != nil {
		log.Printf("Error creating instance: %v", err)
		w.Header().Set("Content-Type", "text/html")
		if err := pages.InstanceError(err.Error()).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html")
	domain := fmt.Sprintf("%s.instol.cloud", resp.Subdomain)
	if err := pages.InstanceCreated(domain).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) HandleListInstances(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	resp, err := h.apiClient.ListInstances(userID)
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
		instances[i] = pages.Instance{
			ID:        inst.ID,
			Subdomain: inst.Subdomain,
			Namespace: fmt.Sprintf("n8n-%s", inst.Subdomain),
			Status:    inst.Status,
			Domain:    fmt.Sprintf("%s.instol.cloud", inst.Subdomain),
			CreatedAt: inst.CreatedAt,
		}
	}

	if err := pages.InstanceListComponent(instances).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) HandleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid instance ID", http.StatusBadRequest)
		return
	}

	if err := h.apiClient.DeleteInstance(id); err != nil {
		log.Printf("Error deleting instance: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
