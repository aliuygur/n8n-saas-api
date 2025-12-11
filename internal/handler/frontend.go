package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/samber/lo"
)

// Home renders the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.HomePage().Render(r.Context(), w))
}

// Dashboard renders the dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := MustGetUser(r.Context())

	// List instances for the user
	instances, err := h.listInstancesInternal(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to fetch instances", slog.Any("error", err))
		http.Error(w, "Failed to load instances", http.StatusInternalServerError)
		return
	}

	instancesView := lo.Map(instances, func(inst Instance, _ int) components.Instance {
		return components.Instance{
			ID:          inst.ID,
			InstanceURL: inst.SubDomain,
			Status:      inst.Status,
			CreatedAt:   inst.CreatedAt.Format(time.RFC3339),
		}
	})

	lo.Must0(components.DashboardPage(instancesView).Render(r.Context(), w))
}

// CreateInstancePage renders the create instance page
func (h *Handler) CreateInstancePage(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.CreateInstancePage().Render(r.Context(), w))
}

// ProvisioningPage renders the provisioning status page
func (h *Handler) ProvisioningPage(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")
	if checkoutID == "" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	lo.Must0(components.ProvisioningStatusPage(checkoutID).Render(r.Context(), w))
}

// GetProvisioningStatus returns the provisioning status for HTMX polling
func (h *Handler) GetProvisioningStatus(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")
	if checkoutID == "" {
		http.Error(w, "checkout_id is required", http.StatusBadRequest)
		return
	}

	// Get checkout session
	session, err := h.getCheckoutSessionByPolarIDInternal(r.Context(), checkoutID)
	if err != nil {
		h.logger.Error("Failed to get checkout session", slog.Any("error", err))
		http.Error(w, "Failed to get checkout session", http.StatusInternalServerError)
		return
	}

	// If completed, get the instance details
	if session.Status == "completed" {
		instance, err := h.getInstanceForComponent(r.Context(), session.InstanceID)
		if err != nil {
			h.logger.Error("Failed to get instance", slog.Any("error", err))
			lo.Must0(components.ProvisioningFailed("Instance not found").Render(r.Context(), w))
			return
		}

		if instance.Status == "deployed" {
			lo.Must0(components.ProvisioningComplete(instance).Render(r.Context(), w))
		} else {
			lo.Must0(components.ProvisioningPending(checkoutID).Render(r.Context(), w))
		}
		return
	}

	if session.Status == "failed" {
		lo.Must0(components.ProvisioningFailed("Provisioning failed. Please try again.").Render(r.Context(), w))
		return
	}

	// Still pending
	lo.Must0(components.ProvisioningPending(checkoutID).Render(r.Context(), w))
}

// DeleteModal renders the delete confirmation modal for an instance
func (h *Handler) DeleteModal(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get instance details to show the subdomain
	instance, err := h.getInstanceInternal(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("failed to get instance", slog.Any("error", err))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Extract subdomain from domain (e.g., "https://myapp.instol.cloud" -> "myapp")
	domain := instance.Subdomain
	subdomain := domain
	if len(domain) > len("https://") && len(domain) > len(".instol.cloud") {
		subdomain = domain[8:]                    // Remove "https://"
		subdomain = subdomain[:len(subdomain)-13] // Remove ".instol.cloud"
	}

	lo.Must0(components.DeleteModal(instanceID, subdomain).Render(r.Context(), w))
}
