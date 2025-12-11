package frontend

import (
	"net/http"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/aliuygur/n8n-saas-api/internal/services/subscription"
	"github.com/samber/lo"
)

// ProvisioningStatusPage renders the provisioning status page
//
//encore:api public raw method=GET path=/provisioning
func (s *Service) ProvisioningStatusPage(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")

	if checkoutID == "" {
		http.Error(w, "checkout_id is required", http.StatusBadRequest)
		return
	}

	lo.Must0(components.ProvisioningStatusPage(checkoutID).Render(r.Context(), w))
}

// ProvisioningStatus returns the current provisioning status via HTMX
//
//encore:api public raw method=GET path=/api/provisioning-status
func (s *Service) ProvisioningStatus(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")

	if checkoutID == "" {
		http.Error(w, "checkout_id is required", http.StatusBadRequest)
		return
	}

	// Get checkout session status from subscription service
	checkoutStatus, err := subscription.GetCheckoutStatus(r.Context(), &subscription.GetCheckoutStatusRequest{
		CheckoutID: checkoutID,
	})
	if err != nil {
		rlog.Error("Failed to get checkout status", "error", err, "checkout_id", checkoutID)
		lo.Must0(components.ProvisioningFailed("Checkout session not found").Render(r.Context(), w))
		return
	}

	// Check status
	switch checkoutStatus.Status {
	case "pending":
		// Still processing
		lo.Must0(components.ProvisioningPending().Render(r.Context(), w))

	case "completed":
		// Get instance to show URL
		instance, err := s.db.GetInstanceBySubdomain(r.Context(), checkoutStatus.Subdomain)
		if err != nil {
			rlog.Error("Failed to get instance", "error", err, "subdomain", checkoutStatus.Subdomain)
			lo.Must0(components.ProvisioningFailed("Instance not found").Render(r.Context(), w))
			return
		}

		if instance.Status == "deployed" {
			// Stop polling by removing hx-trigger attribute
			w.Header().Set("HX-Trigger", "stopPolling")
			lo.Must0(components.ProvisioningComplete(instance.Subdomain).Render(r.Context(), w))
		} else {
			// Still deploying
			lo.Must0(components.ProvisioningPending().Render(r.Context(), w))
		}

	case "failed":
		// Stop polling
		w.Header().Set("HX-Trigger", "stopPolling")
		lo.Must0(components.ProvisioningFailed("Provisioning failed. Please try again.").Render(r.Context(), w))

	default:
		lo.Must0(components.ProvisioningPending().Render(r.Context(), w))
	}
}
