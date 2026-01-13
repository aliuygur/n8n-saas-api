package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

// Account renders the account page with user profile and subscription details
func (h *Handler) Account(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := appctx.GetLogger(ctx)
	user := MustGetUser(ctx)

	// Get user details
	userDetails, err := h.services.GetUser(ctx, user.UserID)
	if err != nil {
		l.Error("Failed to get user details", slog.Any("error", err))
		http.Error(w, "Failed to load user details", http.StatusInternalServerError)
		return
	}

	// Get the user's subscription
	sub, err := h.services.GetUserSubscription(ctx, user.UserID)
	if err != nil {
		l.Error("Failed to get subscription", slog.Any("error", err))
		http.Error(w, "Failed to load subscription", http.StatusInternalServerError)
		return
	}

	// Check if user has a subscription
	if sub == nil {
		http.Error(w, "No subscription found", http.StatusNotFound)
		return
	}

	trialEndsAt := ""
	if sub.TrialEndsAt != nil {
		trialEndsAt = sub.TrialEndsAt.Format(time.RFC3339)
	}

	accountData := components.AccountData{
		User: components.UserAccount{
			ID:        userDetails.ID,
			Email:     userDetails.Email,
			Name:      userDetails.Name,
			CreatedAt: userDetails.CreatedAt.Format(time.RFC3339),
		},
		Subscription: components.Subscription{
			ID:             sub.ID,
			UserID:         sub.UserID,
			ProductID:      sub.ProductID,
			CustomerID:     sub.CustomerID,
			SubscriptionID: sub.SubscriptionID,
			Status:         sub.Status,
			TrialEndsAt:    trialEndsAt,
			CreatedAt:      sub.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      sub.UpdatedAt.Format(time.RFC3339),
			Quantity:       sub.Quantity,
		},
	}

	lo.Must0(components.AccountPage(accountData).Render(ctx, w))
}
