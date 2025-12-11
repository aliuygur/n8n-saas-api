package frontend

import (
	"net/http"

	"encore.dev/rlog"
)

//encore:api public raw method=GET path=/api/checkout-callback
func (s *Service) CheckoutCallback(w http.ResponseWriter, r *http.Request) {
	checkoutID := r.URL.Query().Get("checkout_id")

	rlog.Info("Checkout callback received, redirecting to dashboard", "checkout_id", checkoutID)

	// Note: The actual instance creation and subscription setup is handled by the webhook
	// This callback is just for redirecting the user to a success page
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
