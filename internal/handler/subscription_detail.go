package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

// SubscriptionDetail renders the subscription detail page
func (h *Handler) SubscriptionDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appctx.GetLogger(ctx)
	user := MustGetUser(ctx)

	// Get the user's subscription
	subscriptions, err := h.services.GetUserSubscriptions(ctx, user.UserID)
	if err != nil {
		l.Error("Failed to get subscription", slog.Any("error", err))
		http.Error(w, "Failed to load subscription", http.StatusInternalServerError)
		return
	}

	// Check if user has a subscription
	if len(subscriptions) == 0 {
		http.Error(w, "No subscription found", http.StatusNotFound)
		return
	}

	// Use the first subscription (one subscription per user)
	sub := subscriptions[0]

	trialEndsAt := ""
	if sub.TrialEndsAt != nil {
		trialEndsAt = sub.TrialEndsAt.Format(time.RFC3339)
	}

	subscriptionView := components.Subscription{
		ID:                  sub.ID,
		UserID:              sub.UserID,
		PolarProductID:      sub.PolarProductID,
		PolarCustomerID:     sub.PolarCustomerID,
		PolarSubscriptionID: sub.PolarSubscriptionID,
		Status:              sub.Status,
		TrialEndsAt:         trialEndsAt,
		CreatedAt:           sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           sub.UpdatedAt.Format(time.RFC3339),
	}

	lo.Must0(components.SubscriptionDetailPage(subscriptionView).Render(ctx, w))
}
