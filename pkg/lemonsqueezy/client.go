package lemonsqueezy

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.lemonsqueezy.com/v1"

// Config holds the LemonSqueezy client configuration
type Config struct {
	APIKey        string
	WebhookSecret string
}

// Client is a LemonSqueezy API client
type Client struct {
	apiKey        string
	webhookSecret string
	httpClient    *http.Client
}

// NewClient creates a new LemonSqueezy client
func NewClient(cfg Config) *Client {
	return &Client{
		apiKey:        cfg.APIKey,
		webhookSecret: cfg.WebhookSecret,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// VerifyWebhookSignature verifies the webhook signature
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	if c.webhookSecret == "" {
		return true
	}

	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// SubscriptionItem represents a subscription item
type SubscriptionItem struct {
	ID        int    `json:"id"`
	ProductID int    `json:"product_id"`
	VariantID int    `json:"variant_id"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SubscriptionResponse represents the API response when fetching a subscription
type SubscriptionResponse struct {
	Data struct {
		Attributes struct {
			FirstSubscriptionItem *SubscriptionItem `json:"first_subscription_item"`
		} `json:"attributes"`
	} `json:"data"`
}

// GetSubscription fetches a subscription by ID
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResponse, error) {
	url := fmt.Sprintf("%s/subscriptions/%s", baseURL, subscriptionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var subscription SubscriptionResponse
	if err := json.Unmarshal(body, &subscription); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &subscription, nil
}

// UpdateSubscriptionQuantity updates the quantity of a subscription by subscription ID.
// It fetches the subscription to get the first subscription item ID, then updates its quantity.
func (c *Client) UpdateSubscriptionQuantity(ctx context.Context, subscriptionID string, quantity int32) error {
	subscription, err := c.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if subscription.Data.Attributes.FirstSubscriptionItem == nil {
		return fmt.Errorf("subscription %s has no subscription item", subscriptionID)
	}

	return c.UpdateSubscriptionItemQuantity(ctx, subscription.Data.Attributes.FirstSubscriptionItem.ID, quantity)
}

// UpdateSubscriptionItemQuantity updates the quantity of a subscription item
func (c *Client) UpdateSubscriptionItemQuantity(ctx context.Context, subscriptionItemID int, quantity int32) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "subscription-items",
			"id":   fmt.Sprintf("%d", subscriptionItemID),
			"attributes": map[string]interface{}{
				"quantity": quantity,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	url := fmt.Sprintf("%s/subscription-items/%d", baseURL, subscriptionItemID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "application/vnd.api+json")
}
