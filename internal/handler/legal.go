package handler

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
)

// TermsOfServiceHandler serves the Terms of Service page at /terms
func TermsOfServiceHandler(w http.ResponseWriter, r *http.Request) {
	components.TermsOfServicePage().Render(r.Context(), w)
}

// PrivacyPolicyHandler serves the Privacy Policy page at /privacy
func PrivacyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	components.PrivacyPolicyPage().Render(r.Context(), w)
}

// RefundPolicyHandler serves the Refund Policy page at /refund-policy
func RefundPolicyHandler(w http.ResponseWriter, r *http.Request) {
	components.RefundPolicyPage().Render(r.Context(), w)
}
