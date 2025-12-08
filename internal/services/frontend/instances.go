package frontend

import (
	"net/http"

	"encore.dev"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/auth"
	"github.com/aliuygur/n8n-saas-api/internal/services/frontend/components"
	"github.com/aliuygur/n8n-saas-api/internal/services/provisioning"
	"github.com/aliuygur/n8n-saas-api/internal/services/subscription"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/samber/lo"
)

// CreateInstancePage renders the create instance page
//
//encore:api public raw method=GET path=/create-instance
func (s *Service) CreateInstancePage(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		return
	}

	lo.Must0(components.CreateInstancePage().Render(r.Context(), w))
}

// CreateInstance creates a new instance via HTMX
//
//encore:api public raw method=POST path=/api/create-instance
func (s *Service) CreateInstance(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		return
	}
	user := auth.MustGetUser()

	subdomain := r.FormValue("subdomain")

	rlog.Debug("Creating instance", "user_id", user.ID, "subdomain", subdomain)

	// Check subscription status
	subs, err := subscription.GetSubscriptionByUserID(r.Context(), &subscription.GetSubscriptionByUserIDRequest{
		UserID: user.ID,
	})
	if err != nil {
		rlog.Error("Failed to get subscription status", "error", err)
		lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
		return
	}

	if !subs.CanCreateInstance() {
		lo.Must0(components.CreateInstanceError("You need an active subscription to create an instance.").Render(r.Context(), w))
		return
	}

	// Call provisioning service
	provResp, err := provisioning.CreateInstance(r.Context(), &provisioning.CreateInstanceRequest{
		UserID:    user.ID,
		Subdomain: subdomain,
	})
	if err != nil {
		rlog.Error("Failed to create instance", "error", err)
		lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
		return
	}

	// its first instance, create trial subscription
	if subs.Status == subscription.StatusNone {
		if sub, err := subscription.CreateTrialSubscription(r.Context(), &subscription.CreateTrialSubscriptionRequest{
			UserID: user.ID,
		}); err != nil {
			rlog.Error("Failed to create trial subscription", "error", err)
			lo.Must0(components.CreateInstanceError("Failed to create trial subscription").Render(r.Context(), w))
			return
		} else {
			subs = sub
		}

	} else {
		// Increment seat count in subscription
		if err := subscription.IncrementSeats(r.Context(), &subscription.IncrementSeatsRequest{
			SubscriptionID: subs.ID,
		}); err != nil {
			rlog.Error("Failed to increment seats", "error", err)
			lo.Must0(components.CreateInstanceError("Failed to increment seats").Render(r.Context(), w))
			return
		}
	}

	rlog.Info("Instance created successfully", "instance_id", provResp.InstanceID, "domain", provResp.Domain, "user_id", user.ID)

	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

// CheckSubdomain checks if a subdomain is available via HTMX
//
//encore:api public raw method=POST path=/api/check-subdomain
func (s *Service) CheckSubdomain(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		return
	}

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
	resp, err := provisioning.CheckSubdomainExists(r.Context(), &provisioning.CheckSubdomainExistsRequest{
		Subdomain: subdomain,
	})
	if err != nil {
		rlog.Error("Failed to check subdomain", "error", err)
		lo.Must0(components.SubdomainAvailability(false, "Failed to check subdomain availability").Render(r.Context(), w))
		return
	}
	if resp.Exists {
		lo.Must0(components.SubdomainAvailability(false, "Subdomain is already taken").Render(r.Context(), w))
		return
	}

	lo.Must0(components.SubdomainAvailability(true, "Subdomain is available").Render(r.Context(), w))
}

// DeleteInstance handles instance deletion via HTMX
//
//encore:api public raw method=DELETE path=/instances/:id
func (s *Service) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	if !auth.AuthOnly(r.Context(), w, r) {
		return
	}

	user := auth.MustGetUser()

	instanceID := encore.CurrentRequest().PathParams.Get("id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := provisioning.DeleteInstance(r.Context(), &provisioning.DeleteInstanceRequest{
		InstanceID: instanceID,
	}); err != nil {
		rlog.Error("Failed to delete instance", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: Decrement instance count in subscription

	rlog.Info("Instance deleted successfully", "instance_id", instanceID, "user_id", user.ID)

	// Return success - HTMX will handle removing the element
	w.WriteHeader(http.StatusOK)
}
