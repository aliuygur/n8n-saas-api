package handler

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

// Home renders the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.HomePage().Render(r.Context(), w))
}

// // GetProvisioningStatus returns the provisioning status for HTMX polling
// func (h *Handler) GetProvisioningStatus(w http.ResponseWriter, r *http.Request) {
// 	checkoutID := r.URL.Query().Get("checkout_id")
// 	if checkoutID == "" {
// 		http.Error(w, "checkout_id is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Get checkout session
// 	session, err := h.getCheckoutSessionByPolarIDInternal(r.Context(), checkoutID)
// 	if err != nil {
// 		h.logger.Error("Failed to get checkout session", slog.Any("error", err))
// 		http.Error(w, "Failed to get checkout session", http.StatusInternalServerError)
// 		return
// 	}

// 	// If completed, get the instance details
// 	if session.Status == "completed" {
// 		instance, err := h.getInstanceForComponent(r.Context(), session.InstanceID)
// 		if err != nil {
// 			h.logger.Error("Failed to get instance", slog.Any("error", err))
// 			lo.Must0(components.ProvisioningFailed("Instance not found").Render(r.Context(), w))
// 			return
// 		}

// 		if instance.Status == "deployed" {
// 			lo.Must0(components.ProvisioningComplete(instance).Render(r.Context(), w))
// 		} else {
// 			lo.Must0(components.ProvisioningPending(checkoutID).Render(r.Context(), w))
// 		}
// 		return
// 	}

// 	if session.Status == "failed" {
// 		lo.Must0(components.ProvisioningFailed("Provisioning failed. Please try again.").Render(r.Context(), w))
// 		return
// 	}

// 	// Still pending
// 	lo.Must0(components.ProvisioningPending(checkoutID).Render(r.Context(), w))
// }

// NotFound renders the 404 page
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	lo.Must0(components.NotFoundPage().Render(r.Context(), w))
}
