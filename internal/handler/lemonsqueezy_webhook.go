package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/services"
)

// LemonSqueezyWebhook handles webhooks from Lemon Squeezy
func (h *Handler) LemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	log := appctx.GetLogger(r.Context())

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify webhook signature
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		log.Error("Missing X-Signature header")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	if !h.services.VerifyLemonSqueezySignature(body, signature) {
		log.Error("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook payload
	var payload services.LemonSqueezyWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	log.Info("Received Lemon Squeezy webhook",
		"event", payload.Meta.EventName,
		"subscription_id", payload.Data.ID,
		"test_mode", payload.Meta.TestMode)

	// Handle the webhook event using service layer
	if err := h.services.HandleLemonSqueezyEvent(r.Context(), &payload); err != nil {
		log.Error("Failed to handle webhook event", "error", err)
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
