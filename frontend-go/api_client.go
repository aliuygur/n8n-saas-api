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

func (c *APIClient) CreateInstance(req handlers.CreateInstanceRequest) (*handlers.CreateInstanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/provisioning.ProvisioningService/CreateInstance", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		UserID:    req.UserID,
		Subdomain: req.Subdomain,
		Status:    apiResp.Status,
		URL:       apiResp.Domain,
	}, nil
}

func (c *APIClient) ListInstances(userID string) (*handlers.ListInstancesResponse, error) {
	req := struct {
		UserID string `json:"user_id,omitempty"`
	}{UserID: userID}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/provisioning.ProvisioningService/ListInstances", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var apiResp struct {
		Instances []Instance `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Convert API instances to handler instances
	instances := make([]handlers.Instance, len(apiResp.Instances))
	for i, inst := range apiResp.Instances {
		instances[i] = handlers.Instance{
			ID:        inst.ID,
			UserID:    inst.UserID,
			Subdomain: inst.Subdomain,
			Status:    inst.Status,
			CreatedAt: inst.CreatedAt,
			URL:       inst.URL,
		}
	}

	return &handlers.ListInstancesResponse{
		Instances: instances,
	}, nil
}

func (c *APIClient) DeleteInstance(instanceID int) error {
	req := DeleteInstanceRequest{InstanceID: instanceID}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/provisioning.ProvisioningService/DeleteInstance", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
