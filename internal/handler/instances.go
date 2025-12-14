package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/aliuygur/n8n-saas-api/internal/types"
	"github.com/aliuygur/n8n-saas-api/pkg/domainutils"
	"github.com/samber/lo"
)

// CreateInstance creates a new instance via HTMX
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := MustGetUser(ctx)

	// Acquire database advisory lock to prevent concurrent instance creation
	lo.Must0(h.db.AcquireUserLock(ctx, user.UserID))
	defer lo.Must0(h.db.ReleaseUserLock(ctx, user.UserID))

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
	exists, err := h.checkSubdomainExistsInternal(ctx, subdomain)
	if err != nil {
		h.logger.Error("Failed to check subdomain availability", slog.Any("error", err))
		lo.Must0(components.CreateInstanceError("Failed to check subdomain availability").Render(r.Context(), w))
		return
	}
	if exists {
		h.logger.Error("Subdomain already taken", slog.String("subdomain", subdomain))
		lo.Must0(components.CreateInstanceError("Subdomain is already taken").Render(ctx, w))
		return
	}

	// Generate unique namespace
	namespace, err := h.generateUniqueNamespace(ctx)
	if err != nil {
		lo.Must0(components.CreateInstanceError("Failed to generate unique namespace").Render(ctx, w))
		return
	}

	// Create instance record in database
	instance, err := h.db.CreateInstance(ctx, db.CreateInstanceParams{
		UserID:    user.UserID,
		Namespace: namespace,
		Subdomain: subdomain,
	})
	if err != nil {
		lo.Must0(components.CreateInstanceError("Failed to create instance record").Render(ctx, w))
		return
	}

	// Deploy the instance
	if err := h.deployInstanceInternal(ctx, &DeployInstanceRequest{InstanceID: instance.ID}); err != nil {
		lo.Must0(components.CreateInstanceError("Failed to deploy instance").Render(ctx, w))
		return
	}

	// // Create checkout session for the subscription
	// baseURL := h.config.Server.BaseURL("/")
	// instanceID := uuid.New().String()
	// checkoutResp, err := h.createCheckoutInternal(r.Context(), CreateCheckoutRequest{
	// 	UserID:     user.UserID,
	// 	InstanceID: instanceID,
	// 	Subdomain:  subdomain,
	// 	UserEmail:  user.Email,
	// 	SuccessURL: baseURL + "/provisioning?checkout_id={CHECKOUT_ID}",
	// 	ReturnURL:  baseURL + "/dashboard",
	// })
	// if err != nil {
	// 	h.logger.Error("Failed to create checkout session", slog.Any("error", err))
	// 	lo.Must0(components.CreateInstanceError(err.Error()).Render(r.Context(), w))
	// 	return
	// }

	// h.logger.Info("Checkout session created",
	// 	slog.String("checkout_id", checkoutResp.CheckoutID),
	// 	slog.String("user_id", user.UserID),
	// 	slog.String("subdomain", subdomain))

	// Redirect to Polar checkout page
	w.Header().Set("HX-Redirect", "/dashboard")
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

// deleteInstanceInternal deletes an instance
func (h *Handler) deleteInstanceInternal(ctx context.Context, instanceID string) error {
	// Get instance details
	instance, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Delete from Kubernetes
	if err := h.provisioning.DeleteNamespace(ctx, instance.Namespace); err != nil {
		h.logger.Error("Failed to delete namespace", slog.Any("error", err))
		// Continue with database deletion even if K8s deletion fails
	}

	// Delete Cloudflare DNS record
	if err := h.cloudflare.DeleteDNSRecord(ctx, instance.Subdomain); err != nil {
		h.logger.Error("Failed to delete Cloudflare DNS record", slog.Any("error", err))
		// Continue with database deletion even if Cloudflare deletion fails
	}

	// Soft delete from database
	if err := h.db.DeleteInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to delete instance from database: %w", err)
	}

	return nil
}

// listInstancesInternal lists all instances for a user
func (h *Handler) listInstancesInternal(ctx context.Context, userID string) ([]types.Instance, error) {
	dbInstances, err := h.db.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	instances := lo.Map(dbInstances, func(inst db.Instance, _ int) types.Instance {
		return toDomainInstance(inst)
	})

	return instances, nil
}

func (h *Handler) getInstanceInternal(ctx context.Context, instanceID string) (*types.Instance, error) {
	dbInstance, err := h.db.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

// toDomainInstance maps a db.Instance to a types.Instance (domain layer)
func toDomainInstance(dbInst db.Instance) types.Instance {
	i := types.Instance{
		ID:        dbInst.ID,
		UserID:    dbInst.UserID,
		Status:    dbInst.Status,
		Namespace: dbInst.Namespace,
		Subdomain: dbInst.Subdomain,
		CreatedAt: dbInst.CreatedAt.Time,
		UpdatedAt: dbInst.UpdatedAt.Time,
	}
	if dbInst.DeployedAt.Valid {
		i.DeployedAt = &dbInst.DeployedAt.Time
	}
	if dbInst.DeletedAt.Valid {
		i.DeletedAt = &dbInst.DeletedAt.Time
	}
	return i
}
