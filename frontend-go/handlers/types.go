package handlers

// APIClient interface defines the methods for interacting with the backend API
type APIClient interface {
	CreateInstance(token string, req CreateInstanceRequest) (*CreateInstanceResponse, error)
	ListInstances(token string) (*ListInstancesResponse, error)
	DeleteInstance(token string, id int) error
	// Google OAuth methods
	GetGoogleLoginURL() (string, error)
	HandleGoogleCallback(code, state string) (*GoogleCallbackResponse, error)
}

// Google OAuth types
type GoogleCallbackResponse struct {
	SessionToken string    `json:"session_token"`
	User         *UserInfo `json:"user"`
	ExpiresAt    string    `json:"expires_at"`
}

type UserInfo struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// CreateInstanceRequest represents the request to create a new instance
type CreateInstanceRequest struct {
	Subdomain string `json:"subdomain"`
}

// CreateInstanceResponse represents the response from creating an instance
type CreateInstanceResponse struct {
	ID        int    `json:"id"`
	Subdomain string `json:"subdomain"`
	Status    string `json:"status"`
	URL       string `json:"url"`
}

// Instance represents a single n8n instance
type Instance struct {
	ID        int    `json:"id"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Domain    string `json:"domain"`
	CreatedAt string `json:"created_at"`
}

// ListInstancesResponse represents the response from listing instances
type ListInstancesResponse struct {
	Instances []Instance `json:"instances"`
}
