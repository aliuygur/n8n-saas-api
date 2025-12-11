package frontend

import (
	"net/http"

	"encore.dev/rlog"
)

//encore:api public raw method=GET path=/api/checkout-callback
func (s *Service) CheckoutCallback(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")

	if checkoutID == "" {
		http.Error(w, "checkout_id is required", http.StatusBadRequest)
		return
	}

	rlog.Info("Checkout callback received, redirecting to provisioning status", "checkout_id", checkoutID)

	// Note: The actual instance creation and subscription setup is handled by the webhook
	// This callback redirects the user to the provisioning status page
	http.Redirect(w, r, "/provisioning?checkout_id="+checkoutID, http.StatusSeeOther)
}
