package frontend

import (
	"net/http"

	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/services/subscription"
)

//encore:api public raw method=GET path=/api/checkout-callback
func (s *Service) CheckoutCallback(w http.ResponseWriter, r *http.Request) {

	res, err := subscription.HandleCheckoutCallback(r.Context(), &subscription.HandleCheckoutCallbackRequest{
		CheckoutID: r.URL.Query().Get("checkout_id"),
	})

	if err != nil {
		http.Error(w, "Failed to handle checkout callback", http.StatusInternalServerError)
		return
	}

	rlog.Info("Checkout callback handled successfully", "subscription_id", res.SubscriptionID)
	http.Redirect(w, r, "/create-instance", http.StatusSeeOther)
}
