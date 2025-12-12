package handler

import (
	"log/slog"
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// CreateInstance creates a new instance via HTMX
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	user := MustGetUser(r.Context())

	// Acquire database advisory lock to prevent concurrent instance creation
	lo.Must0(h.db.AcquireUserLock(r.Context(), user.UserID))
	defer lo.Must0(h.db.ReleaseUserLock(r.Context(), user.UserID))

	subdomain := r.FormValue("subdomain")

	h.logger.Debug("Creating checkout session for instance",
		slog.String("user_id", user.UserID),
		slog.String("subdomain", subdomain))

	// Validate subdomain
	if err := domainutils.ValidateSubdomain(subdomain); err != nil {
		h.logger.Error("Invalid subdomain", slog.Any("error", err), slog.String("subdomain", subdomain))
		lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
		return
	}

	// Check if subdomain already exists
	exists, err := h.checkSubdomainExistsInternal(r.Context(), subdomain)
	if err != nil {
		h.logger.Error("Failed to check subdomain availability", slog.Any("error", err))
		lo.Must0(components.CreateInstanceError("Failed to check subdomain availability").Render(r.Context(), w))
		return
	}
	if exists {
		h.logger.Error("Subdomain already taken", slog.String("subdomain", subdomain))
		lo.Must0(components.CreateInstanceError("Subdomain is already taken").Render(r.Context(), w))
		return
	}

	// Create checkout session for the subscription
	baseURL := h.config.Server.BaseURL("/")
	instanceID := uuid.New().String()
	checkoutResp, err := h.createCheckoutInternal(r.Context(), CreateCheckoutRequest{
		UserID:     user.UserID,
		InstanceID: instanceID,
		Subdomain:  subdomain,
		UserEmail:  user.Email,
		SuccessURL: baseURL + "/provisioning?checkout_id={CHECKOUT_ID}",
		ReturnURL:  baseURL + "/dashboard",
	})
	if err != nil {
		h.logger.Error("Failed to create checkout session", slog.Any("error", err))
		lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
		return
	}

	h.logger.Info("Checkout session created",
		slog.String("checkout_id", checkoutResp.CheckoutID),
		slog.String("user_id", user.UserID),
		slog.String("subdomain", subdomain))

	// Redirect to Polar checkout page
	w.Header().Set("HX-Redirect", checkoutResp.CheckoutURL)
	w.WriteHeader(http.StatusOK)
}

// CheckSubdomain checks if a subdomain is available via HTMX
func (h *Handler) CheckSubdomain(w http.ResponseWriter, r *http.Request) {
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
	exists, err := h.checkSubdomainExistsInternal(r.Context(), subdomain)
	if err != nil {
		h.logger.Error("Failed to check subdomain", slog.Any("error", err))
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
	user := MustGetUser(r.Context())

	instanceID := r.PathValue("id")
	if instanceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.deleteInstanceInternal(r.Context(), instanceID); err != nil {
		h.logger.Error("Failed to delete instance", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Delete the subscription associated with this instance
	if err := h.deleteSubscriptionByInstanceIDInternal(r.Context(), instanceID); err != nil {
		h.logger.Error("Failed to delete subscription", slog.Any("error", err))
		// Don't fail the request if subscription deletion fails
	}

	h.logger.Info("Instance deleted successfully",
		slog.String("instance_id", instanceID),
		slog.String("user_id", user.UserID))

	// Return success - HTMX will handle removing the element
	w.WriteHeader(http.StatusOK)
}
