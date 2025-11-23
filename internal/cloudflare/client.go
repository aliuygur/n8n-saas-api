package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"encore.dev/rlog"
)

// Config holds Cloudflare configuration
type Config struct {
	APIToken  string
	TunnelID  string
	AccountID string
	ZoneID    string
}

// Client represents a Cloudflare API client
type Client struct {
	config     Config
	httpClient *http.Client
	baseURL    string
}

// TunnelRoute represents a Cloudflare tunnel route configuration
type TunnelRoute struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
	Path     string `json:"path,omitempty"`
}

// TunnelConfigResponse represents the tunnel configuration API response
type TunnelConfigResponse struct {
	Success  bool  `json:"success"`
	Errors   []any `json:"errors"`
	Messages []any `json:"messages"`
	Result   any   `json:"result"`
}

// NewClient creates a new Cloudflare client with the provided configuration
func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
}

// AddTunnelRoute adds a new route to the existing Cloudflare tunnel
func (c *Client) AddTunnelRoute(ctx context.Context, hostname, serviceURL string) error {
	if c.config.TunnelID == "" {
		return fmt.Errorf("tunnel ID not configured")
	}

	// First, get the current tunnel configuration
	currentConfig, err := c.getTunnelConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current tunnel config: %w", err)
	}

	// Add the new route to the configuration
	newRoute := TunnelRoute{
		Hostname: strings.TrimPrefix(hostname, "https://"),
		Service:  serviceURL,
	}

	updatedConfig := c.addRouteToConfig(currentConfig, newRoute)

	// Update the tunnel configuration
	if err := c.updateTunnelConfig(ctx, updatedConfig); err != nil {
		return fmt.Errorf("failed to update tunnel config: %w", err)
	}

	// Create the CNAME DNS record
	if err := c.CreateCNAMERecord(ctx, hostname); err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	rlog.Info("Successfully added tunnel route and DNS record",
		"hostname", hostname,
		"service", serviceURL,
		"tunnel_id", c.config.TunnelID)

	return nil
}

// getTunnelConfig retrieves the current tunnel configuration
func (c *Client) getTunnelConfig(ctx context.Context) (map[string]any, error) {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/configurations",
		c.baseURL, c.config.AccountID, c.config.TunnelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error: %s", string(body))
	}

	var response struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("cloudflare API returned success=false")
	}

	// Extract just the config portion from the result
	if config, ok := response.Result["config"].(map[string]any); ok {
		return config, nil
	}

	// If there's no config field, return empty config
	return make(map[string]any), nil
}

// updateTunnelConfig updates the tunnel configuration
func (c *Client) updateTunnelConfig(ctx context.Context, config map[string]any) error {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/configurations",
		c.baseURL, c.config.AccountID, c.config.TunnelID)

	configJSON, err := json.Marshal(map[string]any{"config": config})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(configJSON))
	if err != nil {
		return err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare API error: %s", string(body))
	}

	var response TunnelConfigResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("cloudflare API returned success=false: %v", response.Errors)
	}

	return nil
}

// addRouteToConfig adds a new route to the existing configuration
func (c *Client) addRouteToConfig(currentConfig map[string]any, newRoute TunnelRoute) map[string]any {
	// If no config exists, create a basic one
	if currentConfig == nil {
		currentConfig = make(map[string]any)
	}

	// Get or create ingress rules
	var ingress []map[string]any
	if existingIngress, exists := currentConfig["ingress"]; exists {
		if ingressSlice, ok := existingIngress.([]any); ok {
			for _, rule := range ingressSlice {
				if ruleMap, ok := rule.(map[string]any); ok {
					ingress = append(ingress, ruleMap)
				}
			}
		}
	}

	// Add the new route (before the catch-all rule if it exists)
	newIngressRule := map[string]any{
		"hostname": newRoute.Hostname,
		"service":  newRoute.Service,
	}

	// Check if the route already exists to avoid duplicates
	for _, rule := range ingress {
		if hostname, hasHostname := rule["hostname"]; hasHostname && hostname == newRoute.Hostname {
			rlog.Debug("Route already exists, skipping", "hostname", newRoute.Hostname)
			return currentConfig
		}
	}

	// Find if there's a catch-all rule (no hostname or empty hostname) and insert before it
	var catchAllIndex = -1
	for i, rule := range ingress {
		hostname, hasHostname := rule["hostname"]
		if !hasHostname || hostname == "" {
			catchAllIndex = i
			break
		}
	}

	if catchAllIndex >= 0 {
		// Insert before catch-all (keep all existing routes)
		ingress = append(ingress[:catchAllIndex], append([]map[string]any{newIngressRule}, ingress[catchAllIndex:]...)...)
	} else {
		// No catch-all found, append the new route and add a default catch-all rule
		ingress = append(ingress, newIngressRule)
		ingress = append(ingress, map[string]any{
			"service": "http_status:404",
		})
	}

	currentConfig["ingress"] = ingress
	return currentConfig
}

// addHeaders adds required headers for Cloudflare API requests
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
}

// RemoveTunnelRoute removes a route from the Cloudflare tunnel
func (c *Client) RemoveTunnelRoute(ctx context.Context, hostname string) error {
	// Get current configuration
	currentConfig, err := c.getTunnelConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current tunnel config: %w", err)
	}

	// Remove the route from configuration
	updatedConfig := c.removeRouteFromConfig(currentConfig, hostname)

	// Update the tunnel configuration
	if err := c.updateTunnelConfig(ctx, updatedConfig); err != nil {
		return fmt.Errorf("failed to update tunnel config: %w", err)
	}

	// Delete the DNS record
	if err := c.DeleteDNSRecord(ctx, hostname); err != nil {
		// Log the error but don't fail the entire operation
		rlog.Error("Failed to delete DNS record", "error", err, "hostname", hostname)
	}

	rlog.Info("Successfully removed tunnel route and DNS record", "hostname", hostname)
	return nil
}

// removeRouteFromConfig removes a route by hostname from the configuration
func (c *Client) removeRouteFromConfig(currentConfig map[string]any, hostname string) map[string]any {
	if currentConfig == nil {
		return currentConfig
	}

	existingIngress, exists := currentConfig["ingress"]
	if !exists {
		return currentConfig
	}

	ingressSlice, ok := existingIngress.([]any)
	if !ok {
		return currentConfig
	}

	var filteredIngress []map[string]any
	for _, rule := range ingressSlice {
		if ruleMap, ok := rule.(map[string]any); ok {
			if ruleHostname, hasHostname := ruleMap["hostname"]; !hasHostname || ruleHostname != hostname {
				filteredIngress = append(filteredIngress, ruleMap)
			}
		}
	}

	currentConfig["ingress"] = filteredIngress
	return currentConfig
}

// ResetTunnelConfig resets the tunnel configuration to a clean state with only a catch-all rule
func (c *Client) ResetTunnelConfig(ctx context.Context) error {
	cleanConfig := map[string]any{
		"ingress": []map[string]any{
			{
				"service": "http_status:404",
			},
		},
		"warp-routing": map[string]any{
			"enabled": false,
		},
	}

	if err := c.updateTunnelConfig(ctx, cleanConfig); err != nil {
		return fmt.Errorf("failed to reset tunnel config: %w", err)
	}

	rlog.Info("Successfully reset tunnel configuration")
	return nil
}

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl,omitempty"`
}

// DNSRecordResponse represents the DNS record API response
type DNSRecordResponse struct {
	Success  bool        `json:"success"`
	Errors   []any       `json:"errors"`
	Messages []any       `json:"messages"`
	Result   []DNSRecord `json:"result"`
}

// CreateCNAMERecord creates a CNAME record pointing to the tunnel
func (c *Client) CreateCNAMERecord(ctx context.Context, hostname string) error {
	if c.config.ZoneID == "" {
		return fmt.Errorf("zone ID not configured")
	}
	if c.config.TunnelID == "" {
		return fmt.Errorf("tunnel ID not configured")
	}

	// Clean hostname (remove https:// if present)
	hostname = strings.TrimPrefix(hostname, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")

	// The CNAME should point to the tunnel ID
	tunnelTarget := fmt.Sprintf("%s.cfargotunnel.com", c.config.TunnelID)

	// Check if DNS record already exists
	existingRecord, err := c.getDNSRecord(ctx, hostname)
	if err != nil {
		return fmt.Errorf("failed to check existing DNS record: %w", err)
	}

	if existingRecord != nil {
		rlog.Info("DNS record already exists", "hostname", hostname, "record_id", existingRecord.ID)
		return nil
	}

	// Create the DNS record
	url := fmt.Sprintf("%s/zones/%s/dns_records", c.baseURL, c.config.ZoneID)

	record := DNSRecord{
		Type:    "CNAME",
		Name:    hostname,
		Content: tunnelTarget,
		Proxied: true,
		TTL:     1, // Auto TTL when proxied
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(recordJSON))
	if err != nil {
		return err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare API error: %s", string(body))
	}

	var response struct {
		Success bool      `json:"success"`
		Errors  []any     `json:"errors"`
		Result  DNSRecord `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("cloudflare API returned success=false: %v", response.Errors)
	}

	rlog.Info("Successfully created DNS record",
		"hostname", hostname,
		"target", tunnelTarget,
		"record_id", response.Result.ID)

	return nil
}

// getDNSRecord retrieves a DNS record by name
func (c *Client) getDNSRecord(ctx context.Context, hostname string) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?name=%s", c.baseURL, c.config.ZoneID, hostname)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error: %s", string(body))
	}

	var response DNSRecordResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("cloudflare API returned success=false: %v", response.Errors)
	}

	if len(response.Result) == 0 {
		return nil, nil
	}

	return &response.Result[0], nil
}

// DeleteDNSRecord deletes a DNS record by hostname
func (c *Client) DeleteDNSRecord(ctx context.Context, hostname string) error {
	if c.config.ZoneID == "" {
		return fmt.Errorf("zone ID not configured")
	}

	// Clean hostname
	hostname = strings.TrimPrefix(hostname, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")

	// Get the DNS record ID
	record, err := c.getDNSRecord(ctx, hostname)
	if err != nil {
		return fmt.Errorf("failed to get DNS record: %w", err)
	}

	if record == nil {
		rlog.Info("DNS record not found, nothing to delete", "hostname", hostname)
		return nil
	}

	// Delete the DNS record
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, c.config.ZoneID, record.ID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare API error: %s", string(body))
	}

	var response struct {
		Success bool  `json:"success"`
		Errors  []any `json:"errors"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("cloudflare API returned success=false: %v", response.Errors)
	}

	rlog.Info("Successfully deleted DNS record", "hostname", hostname, "record_id", record.ID)
	return nil
}
