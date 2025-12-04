package handlers

// APIClient interface defines the methods for interacting with the backend API
type APIClient interface {
	CreateInstance(req CreateInstanceRequest) (*CreateInstanceResponse, error)
	ListInstances(userID string) (*ListInstancesResponse, error)
	DeleteInstance(id int) error
}

// CreateInstanceRequest represents the request to create a new instance
type CreateInstanceRequest struct {
	UserID    string `json:"user_id"`
	Subdomain string `json:"subdomain"`
}

// CreateInstanceResponse represents the response from creating an instance
type CreateInstanceResponse struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	Subdomain string `json:"subdomain"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

// Instance represents a single n8n instance
type Instance struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	Subdomain string `json:"subdomain"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

// ListInstancesResponse represents the response from listing instances
type ListInstancesResponse struct {
	Instances []Instance `json:"instances"`
}
