package frontend

import (
	"net/http"
	"time"

	"encore.dev"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/auth"
	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
	"github.com/samber/lo"
)

// Dashboard renders the dashboard page
//
//encore:api public raw method=GET path=/dashboard
func (s *Service) Dashboard(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		rlog.Info("unauthenticated access to dashboard")
		return
	}

	resp, err := provisioning.ListInstances(r.Context(), &provisioning.ListInstancesRequest{
		UserID: lo.Must(auth.GetUserID()),
	})
	if err != nil {
		rlog.Error("failed to fetch instances", "error", err)
		http.Error(w, "Failed to load instances", http.StatusInternalServerError)
	}

	instances := lo.Map(resp.Instances, func(inst *provisioning.Instance, _ int) components.Instance {
		return components.Instance{
			ID:          inst.ID,
			InstanceURL: inst.SubDomain,
			Status:      inst.Status,
			CreatedAt:   inst.CreatedAt.Format(time.RFC3339),
		}
	})

	lo.Must0(components.DashboardPage(instances).Render(r.Context(), w))
}

// DeleteModal renders the delete confirmation modal for an instance
//
//encore:api public raw method=GET path=/api/delete-modal/:id
func (s *Service) DeleteModal(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		return
	}

	instanceID := encore.CurrentRequest().PathParams.Get("id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get instance details to show the subdomain
	instance, err := provisioning.GetInstance(r.Context(), &provisioning.GetInstanceRequest{
		InstanceID: instanceID,
	})
	if err != nil {
		rlog.Error("failed to get instance", "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Extract subdomain from domain (e.g., "https://myapp.instol.cloud" -> "myapp")
	domain := instance.SubDomain
	subdomain := domain
	if len(domain) > len("https://") && len(domain) > len(".instol.cloud") {
		subdomain = domain[8:]                    // Remove "https://"
		subdomain = subdomain[:len(subdomain)-13] // Remove ".instol.cloud"
	}

	lo.Must0(components.DeleteModal(instanceID, subdomain).Render(r.Context(), w))
}
