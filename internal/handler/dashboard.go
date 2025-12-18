package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	"github.com/samber/lo"
)

// Dashboard renders the dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := MustGetUser(r.Context())

	// List instances for the user
	instances, err := h.services.GetInstancesByUser(r.Context(), user.UserID)
	if err != nil {
		appctx.GetLogger(r.Context()).Error("failed to fetch instances", slog.Any("error", err))
		http.Error(w, "Failed to load instances", http.StatusInternalServerError)
		return
	}

	instancesView := lo.Map(instances, func(inst services.Instance, _ int) components.Instance {
		return components.Instance{
			ID:          inst.ID,
			InstanceURL: inst.GetInstanceURL(),
			Status:      inst.Status,
			Subdomain:   inst.Subdomain,
			CreatedAt:   inst.CreatedAt.Format(time.RFC3339),
		}
	})

	lo.Must0(components.DashboardPage(instancesView).Render(r.Context(), w))
}
