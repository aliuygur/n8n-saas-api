package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aliuygur/n8n-saas-api/frontend-go/handlers"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient() *APIClient {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4000"
	}

	return &APIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Google OAuth types
type GoogleLoginResponse struct {
	AuthURL string `json:"auth_url"`
}

type GoogleCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type Instance struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	Subdomain string `json:"subdomain"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	URL       string `json:"url,omitempty"`
}

type DeleteInstanceRequest struct {
	InstanceID int `json:"instance_id"`
}

func (c *APIClient) CreateInstance(token string, req handlers.CreateInstanceRequest) (*handlers.CreateInstanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/instances", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("%s", errResp.Message)
		}
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var apiResp struct {
		InstanceID int    `json:"instance_id"`
		Status     string `json:"status"`
		Domain     string `json:"domain"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Convert API response to handler response
	return &handlers.CreateInstanceResponse{
		ID:        apiResp.InstanceID,
		Subdomain: req.Subdomain,
		Status:    apiResp.Status,
		URL:       apiResp.Domain,
	}, nil
}

func (c *APIClient) ListInstances(token string) (*handlers.ListInstancesResponse, error) {
	httpReq, err := http.NewRequest("GET", c.baseURL+"/api/instances", nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var apiResp struct {
		Instances []struct {
			ID         int    `json:"id"`
			Status     string `json:"status"`
			Domain     string `json:"domain"`
			Namespace  string `json:"namespace"`
			ServiceURL string `json:"service_url"`
			CreatedAt  string `json:"created_at"`
			DeployedAt string `json:"deployed_at"`
			Details    string `json:"details"`
		} `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Convert API instances to handler instances
	instances := make([]handlers.Instance, len(apiResp.Instances))
	for i, inst := range apiResp.Instances {
		instances[i] = handlers.Instance{
			ID:        inst.ID,
			Namespace: inst.Namespace,
			Status:    inst.Status,
			Domain:    inst.Domain,
			CreatedAt: inst.CreatedAt,
		}
	}

	return &handlers.ListInstancesResponse{
		Instances: instances,
	}, nil
}

func (c *APIClient) DeleteInstance(token string, instanceID int) error {
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/instances/%d", c.baseURL, instanceID), nil)
	if err != nil {
		return err
	}
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	return nil
}

// Google OAuth methods

func (c *APIClient) GetGoogleLoginURL() (string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/auth/google/login")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get Google login URL: %s", resp.Status)
	}

	var result GoogleLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AuthURL, nil
}

func (c *APIClient) HandleGoogleCallback(code, state string) (*handlers.GoogleCallbackResponse, error) {
	reqBody := GoogleCallbackRequest{
		Code:  code,
		State: state,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/auth/google/callback", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google callback failed: %s", resp.Status)
	}

	var result handlers.GoogleCallbackResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
