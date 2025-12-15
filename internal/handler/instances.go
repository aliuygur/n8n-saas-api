package handler

import (
	"log/slog"
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/appreq"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/samber/lo"
)

// CreateInstance creates a new instance via HTMX
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appreq.GetLogger(ctx)
	user := MustGetUser(ctx)

	subdomain := r.FormValue("subdomain")

	instance, err := h.services.CreateInstance(ctx, services.CreateInstanceParams{
		UserID:    user.UserID,
		Subdomain: subdomain,
	})
	if err != nil {
		appreq.GetLogger(r.Context()).Error("Failed to create instance", slog.Any("error", err))
		lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
		return
	}

	l.Info("Instance created successfully",
		slog.String("instance_id", instance.ID),
		slog.String("user_id", user.UserID),
		slog.String("subdomain", subdomain))

	// Redirect to provisioning page to wait for instance to be ready
	w.Header().Set("HX-Redirect", "/provision?instance_id="+instance.ID)
	w.WriteHeader(http.StatusOK)
}

// CheckSubdomain checks if a subdomain is available via HTMX
func (h *Handler) CheckSubdomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appreq.GetLogger(ctx)

	subdomain := r.FormValue("subdomain")
	if subdomain == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate subdomain
	if err := domainutils.ValidateSubdomain(subdomain); err != nil {
		lo.Must0(components.SubdomainAvailability(false, err.Error()).Render(r.Context(), w))
		return
	}

	// Check if subdomain already exists
	exists, err := h.services.CheckSubdomainExists(ctx, subdomain)
	if err != nil {
		l.Error("Failed to check subdomain", slog.Any("error", err))
		lo.Must0(components.SubdomainAvailability(false, "Failed to check subdomain availability").Render(r.Context(), w))
		return
	}
	if exists {
		lo.Must0(components.SubdomainAvailability(false, "Subdomain is already taken").Render(r.Context(), w))
		return
	}

	lo.Must0(components.SubdomainAvailability(true, "Subdomain is available").Render(r.Context(), w))
}

// DeleteInstance handles instance deletion via HTMX
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appreq.GetLogger(ctx)
	user := MustGetUser(ctx)

	instanceID := r.PathValue("id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.services.DeleteInstance(ctx, services.DeleteInstanceParams{
		UserID:     user.UserID,
		InstanceID: instanceID,
	}); err != nil {
		l.Error("Failed to delete instance", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	l.Info("Instance deleted successfully",
		slog.String("instance_id", instanceID),
		slog.String("user_id", user.UserID))

	// Return success - HTMX will handle removing the element
	w.WriteHeader(http.StatusOK)
}

// CheckInstanceStatus checks if the instance URL is active via HTMX polling
func (h *Handler) CheckInstanceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appreq.GetLogger(ctx)
	user := MustGetUser(ctx)

	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the instance
	instance, err := h.services.GetInstanceByID(ctx, instanceID)
	if err != nil {
		l.Error("Failed to get instance", slog.Any("error", err))
		lo.Must0(components.ProvisioningFailed("Instance not found").Render(ctx, w))
		return
	}

	// Verify the instance belongs to the user
	if instance.UserID != user.UserID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Check if instance URL is active
	isActive, err := h.services.CheckInstanceURLActive(ctx, instance.GetInstanceURL())
	if err != nil {
		l.Error("Failed to check instance URL", slog.Any("error", err))
		// Continue polling - don't show error yet
		lo.Must0(components.ProvisioningPending(instanceID).Render(ctx, w))
		return
	}

	if isActive {
		// Instance is ready!
		componentInstance := &components.Instance{
			ID:          instance.ID,
			InstanceURL: instance.GetInstanceURL(),
			Status:      instance.Status,
			CreatedAt:   instance.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		lo.Must0(components.ProvisioningComplete(componentInstance).Render(ctx, w))
		return
	}

	// Still provisioning
	lo.Must0(components.ProvisioningPending(instanceID).Render(ctx, w))
}
