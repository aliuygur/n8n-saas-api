package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/gke"
)

type CreateInstanceRequest struct {
	Name                string `json:"name"`
	UserID              string `json:"user_id"`
	Namespace           string `json:"namespace"`
	Domain              string `json:"domain"`
	WorkerReplicas      int    `json:"worker_replicas,omitempty"`
	MainCPURequest      string `json:"main_cpu_request,omitempty"`
	MainMemoryRequest   string `json:"main_memory_request,omitempty"`
	WorkerCPURequest    string `json:"worker_cpu_request,omitempty"`
	WorkerMemoryRequest string `json:"worker_memory_request,omitempty"`
	PostgresStorageSize string `json:"postgres_storage_size,omitempty"`
	N8NStorageSize      string `json:"n8n_storage_size,omitempty"`
}

type CreateInstanceResponse struct {
	InstanceID   int    `json:"instance_id"`
	DeploymentID int    `json:"deployment_id"`
	Status       string `json:"status"`
}

type ScaleRequest struct {
	InstanceID     int `json:"instance_id"`
	WorkerReplicas int `json:"worker_replicas"`
}

// Namespace suggestion API
type NamespaceSuggestionRequest struct {
	UserID string `json:"user_id"`
}

type NamespaceSuggestionResponse struct {
	SuggestedNamespace string `json:"suggested_namespace"`
	Valid              bool   `json:"valid"`
	Message            string `json:"message,omitempty"`
}

//encore:api private
func (s *Service) SuggestNamespace(ctx context.Context, req *NamespaceSuggestionRequest) (*NamespaceSuggestionResponse, error) {
	if req.UserID == "" {
		return &NamespaceSuggestionResponse{
			Valid:   false,
			Message: "user_id is required",
		}, nil
	}

	suggested := GenerateNamespaceName(req.UserID)
	err := validateNamespace(suggested)

	return &NamespaceSuggestionResponse{
		SuggestedNamespace: suggested,
		Valid:              err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "Valid namespace for n8n instance isolation"
		}(),
	}, nil
}

//encore:api private
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	// Validate required fields
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required for customer isolation")
	}
	if err := validateNamespace(req.Namespace); err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// Set defaults
	if req.WorkerReplicas == 0 {
		req.WorkerReplicas = 1
	}
	if req.MainCPURequest == "" {
		req.MainCPURequest = "500m"
	}
	if req.MainMemoryRequest == "" {
		req.MainMemoryRequest = "512Mi"
	}
	if req.WorkerCPURequest == "" {
		req.WorkerCPURequest = "250m"
	}
	if req.WorkerMemoryRequest == "" {
		req.WorkerMemoryRequest = "256Mi"
	}
	if req.PostgresStorageSize == "" {
		req.PostgresStorageSize = "10Gi"
	}
	if req.N8NStorageSize == "" {
		req.N8NStorageSize = "5Gi"
	}

	// Generate secure credentials
	encryptionKey, err := generateSecureKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	dbPassword, err := generateSecureKey(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate database password: %w", err)
	}

	// Create database record
	queries := db.New(s.db)

	instance, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
		Name:                req.Name,
		UserID:              req.UserID,
		GkeClusterName:      s.config.DefaultClusterName,
		GkeProjectID:        s.config.DefaultProjectID,
		GkeZone:             s.config.DefaultZone,
		Namespace:           req.Namespace,
		Domain:              req.Domain,
		WorkerReplicas:      int32(req.WorkerReplicas),
		MainCpuRequest:      req.MainCPURequest,
		MainMemoryRequest:   req.MainMemoryRequest,
		WorkerCpuRequest:    req.WorkerCPURequest,
		WorkerMemoryRequest: req.WorkerMemoryRequest,
		PostgresStorageSize: req.PostgresStorageSize,
		N8nStorageSize:      req.N8NStorageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Create deployment record
	deployment, err := queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		InstanceID: instance.ID,
		Operation:  "deploy",
		Details:    []byte(`{"encryption_key":"***","db_password":"***"}`),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Start async deployment
	go s.deployInstanceAsync(context.Background(), instance, deployment.ID, encryptionKey, dbPassword)

	return &CreateInstanceResponse{
		InstanceID:   int(instance.ID),
		DeploymentID: int(deployment.ID),
		Status:       instance.Status,
	}, nil
}

// Get Instance API
type GetInstanceRequest struct {
	InstanceID int `json:"instance_id"`
}

type InstanceStatus struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Domain     string     `json:"domain"`
	CreatedAt  time.Time  `json:"created_at"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
	Details    string     `json:"details,omitempty"` // JSON string instead of interface{}
}

//encore:api private
func (s *Service) GetInstance(ctx context.Context, req *GetInstanceRequest) (*InstanceStatus, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	response := &InstanceStatus{
		ID:        int(instance.ID),
		Name:      instance.Name,
		Status:    instance.Status,
		CreatedAt: instance.CreatedAt.Time,
		Domain:    instance.Domain,
	}

	if instance.DeployedAt.Valid {
		response.DeployedAt = &instance.DeployedAt.Time
	}

	// Get live status from Kubernetes if deployed
	if instance.Status == "deployed" && instance.Namespace != "" {
		if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err == nil {
			if details, err := s.gke.GetInstanceStatus(ctx, instance.Namespace); err == nil {
				if detailsJSON, err := json.Marshal(details); err == nil {
					response.Details = string(detailsJSON)
				}
			}
		}
	}

	return response, nil
}

// List Instances API
type ListInstancesRequest struct {
	UserID string `json:"user_id"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type ListInstancesResponse struct {
	Instances []*InstanceStatus `json:"instances"`
}

//encore:api private
func (s *Service) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	queries := db.New(s.db)

	var instances []db.Instance
	var err error

	if req.UserID != "" {
		instances, err = queries.ListInstancesByUser(ctx, req.UserID)
	} else {
		instances, err = queries.ListAllInstances(ctx, db.ListAllInstancesParams{
			Limit:  int32(limit),
			Offset: int32(req.Offset),
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	result := make([]*InstanceStatus, len(instances))
	for i, instance := range instances {
		result[i] = &InstanceStatus{
			ID:        int(instance.ID),
			Name:      instance.Name,
			Status:    instance.Status,
			Domain:    instance.Domain,
			CreatedAt: instance.CreatedAt.Time,
		}

		if instance.DeployedAt.Valid {
			result[i].DeployedAt = &instance.DeployedAt.Time
		}
	}

	return &ListInstancesResponse{Instances: result}, nil
}

// Scale Instance API
type ScaleInstanceRequest struct {
	InstanceID int `json:"instance_id"`
	ScaleRequest
}

//encore:api private
func (s *Service) ScaleInstance(ctx context.Context, req *ScaleInstanceRequest) (*CreateInstanceResponse, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return nil, fmt.Errorf("instance must be deployed to scale")
	}

	// Update database
	_, err = queries.UpdateInstanceResources(ctx, db.UpdateInstanceResourcesParams{
		ID:                  instance.ID,
		WorkerReplicas:      int32(req.WorkerReplicas),
		MainCpuRequest:      instance.MainCpuRequest,
		MainMemoryRequest:   instance.MainMemoryRequest,
		WorkerCpuRequest:    instance.WorkerCpuRequest,
		WorkerMemoryRequest: instance.WorkerMemoryRequest,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	// Create deployment record
	deployment, err := queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		InstanceID: instance.ID,
		Operation:  "scale",
		Details:    []byte(fmt.Sprintf(`{"worker_replicas":%d}`, req.WorkerReplicas)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Start async scaling
	go s.scaleInstanceAsync(context.Background(), instance, deployment.ID, int32(req.WorkerReplicas))

	return &CreateInstanceResponse{
		InstanceID:   int(instance.ID),
		DeploymentID: int(deployment.ID),
		Status:       "scaling",
	}, nil
}

// Delete Instance API
type DeleteInstanceRequest struct {
	InstanceID int `json:"instance_id"`
}

//encore:api private
func (s *Service) DeleteInstance(ctx context.Context, req *DeleteInstanceRequest) error {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Create deletion deployment record
	deployment, err := queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		InstanceID: instance.ID,
		Operation:  "delete",
		Details:    []byte(`{}`),
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Start async deletion
	go s.deleteInstanceAsync(context.Background(), instance, deployment.ID)

	return nil
}

// Get Instance Deployments API
type GetInstanceDeploymentsRequest struct {
	InstanceID int `json:"instance_id"`
	Limit      int `json:"limit,omitempty"`
	Offset     int `json:"offset,omitempty"`
}

type GetInstanceDeploymentsResponse struct {
	Deployments []*DeploymentInfo `json:"deployments"`
}

type DeploymentInfo struct {
	ID           int        `json:"id"`
	InstanceID   int        `json:"instance_id"`
	Operation    string     `json:"operation"`
	Status       string     `json:"status"`
	Details      string     `json:"details,omitempty"` // JSON string instead of interface{}
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

//encore:api private
func (s *Service) GetInstanceDeployments(ctx context.Context, req *GetInstanceDeploymentsRequest) (*GetInstanceDeploymentsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	queries := db.New(s.db)

	deployments, err := queries.ListDeploymentsByInstance(ctx, db.ListDeploymentsByInstanceParams{
		InstanceID: int32(req.InstanceID),
		Limit:      int32(limit),
		Offset:     int32(req.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	result := make([]*DeploymentInfo, len(deployments))
	for i, deployment := range deployments {
		result[i] = &DeploymentInfo{
			ID:         int(deployment.ID),
			InstanceID: int(deployment.InstanceID),
			Operation:  deployment.Operation,
			Status:     deployment.Status,
			StartedAt:  deployment.StartedAt.Time,
		}

		if deployment.ErrorMessage != "" {
			result[i].ErrorMessage = &deployment.ErrorMessage
		}

		if deployment.CompletedAt.Valid {
			result[i].CompletedAt = &deployment.CompletedAt.Time
		}

		// Parse details JSON if available
		if len(deployment.Details) > 0 {
			result[i].Details = string(deployment.Details)
		}
	}

	return &GetInstanceDeploymentsResponse{Deployments: result}, nil
}

func (s *Service) deployInstanceAsync(ctx context.Context, instance db.Instance, deploymentID int32, encryptionKey, dbPassword string) {
	queries := db.New(s.db)

	log.Printf("Starting deployment: instance_id=%d deployment_id=%d", instance.ID, deploymentID)

	// Update deployment status to running
	_, err := queries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID:           deploymentID,
		Status:       "running",
		ErrorMessage: "",
	})
	if err != nil {
		log.Printf("Failed to update deployment status: %v", err)
		return
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to connect to cluster: %v", err))
		return
	}

	// Deploy n8n instance
	n8nInstance := gke.N8NInstance{
		Name:                instance.Name,
		Namespace:           instance.Namespace,
		Domain:              instance.Domain,
		WorkerReplicas:      instance.WorkerReplicas,
		MainCPURequest:      instance.MainCpuRequest,
		MainMemoryRequest:   instance.MainMemoryRequest,
		WorkerCPURequest:    instance.WorkerCpuRequest,
		WorkerMemoryRequest: instance.WorkerMemoryRequest,
		PostgresStorageSize: instance.PostgresStorageSize,
		N8NStorageSize:      instance.N8nStorageSize,
		EncryptionKey:       encryptionKey,
		DatabasePassword:    dbPassword,
	}

	if err := s.gke.DeployN8NInstance(ctx, n8nInstance); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to deploy n8n: %v", err))
		return
	}

	// Mark as deployed
	_, err = queries.UpdateInstanceDeployed(ctx, db.UpdateInstanceDeployedParams{
		ID:     instance.ID,
		Status: "deployed",
	})
	if err != nil {
		log.Printf("Failed to update instance status: %v", err)
	}

	_, err = queries.UpdateDeploymentCompleted(ctx, deploymentID)
	if err != nil {
		log.Printf("Failed to mark deployment completed: %v", err)
	}

	log.Printf("Deployment completed successfully: instance_id=%d", instance.ID)
}

func (s *Service) scaleInstanceAsync(ctx context.Context, instance db.Instance, deploymentID int32, replicas int32) {
	queries := db.New(s.db)

	log.Printf("Starting scaling: instance_id=%d replicas=%d", instance.ID, replicas)

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to connect to cluster: %v", err))
		return
	}

	// Scale workers
	if err := s.gke.ScaleWorkers(ctx, instance.Namespace, replicas); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to scale workers: %v", err))
		return
	}

	// Mark as completed
	_, err := queries.UpdateDeploymentCompleted(ctx, deploymentID)
	if err != nil {
		log.Printf("Failed to mark deployment completed: %v", err)
	}

	log.Printf("Scaling completed successfully: instance_id=%d", instance.ID)
}

func (s *Service) deleteInstanceAsync(ctx context.Context, instance db.Instance, deploymentID int32) {
	queries := db.New(s.db)

	log.Printf("Starting deletion: instance_id=%d", instance.ID)

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to connect to cluster: %v", err))
		return
	}

	// Delete n8n instance
	if err := s.gke.DeleteN8NInstance(ctx, instance.Namespace); err != nil {
		s.markDeploymentFailed(ctx, queries, deploymentID, fmt.Sprintf("Failed to delete n8n: %v", err))
		return
	}

	// Soft delete from database
	_, err := queries.SoftDeleteInstance(ctx, instance.ID)
	if err != nil {
		log.Printf("Failed to soft delete instance: %v", err)
	}

	// Mark deployment as completed
	_, err = queries.UpdateDeploymentCompleted(ctx, deploymentID)
	if err != nil {
		log.Printf("Failed to mark deployment completed: %v", err)
	}

	log.Printf("Deletion completed successfully: instance_id=%d", instance.ID)
}

// validateNamespace validates that the namespace follows Kubernetes naming conventions
// and provides good n8n instance isolation
func validateNamespace(namespace string) error {
	if len(namespace) == 0 {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(namespace) > 63 {
		return fmt.Errorf("namespace cannot be longer than 63 characters")
	}

	// Kubernetes namespace naming rules:
	// - Must be lowercase alphanumeric or hyphens
	// - Must start and end with alphanumeric
	validNamespace := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !validNamespace.MatchString(namespace) {
		return fmt.Errorf("namespace must contain only lowercase letters, numbers, and hyphens, and start/end with alphanumeric character")
	}

	// Prevent using system namespaces
	systemNamespaces := []string{"kube-system", "kube-public", "kube-node-lease", "default"}
	for _, sysNs := range systemNamespaces {
		if namespace == sysNs {
			return fmt.Errorf("cannot use system namespace: %s", namespace)
		}
	}

	return nil
}

// GenerateNamespaceName creates a suggested namespace name for n8n instance isolation
// Format: n8n-instance-{sanitized-userID} (truncated and sanitized if needed)
func GenerateNamespaceName(userID string) string {
	// Sanitize userID to be kubernetes-compliant
	sanitized := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(strings.ToLower(userID), "")

	// Create namespace with n8n-instance prefix for clarity
	namespace := "n8n-instance-" + sanitized

	// Truncate if too long (leaving room for suffix if needed)
	if len(namespace) > 58 {
		namespace = namespace[:58]
	}

	// Ensure it ends with alphanumeric
	if len(namespace) > 0 && namespace[len(namespace)-1] == '-' {
		namespace = namespace[:len(namespace)-1] + "0"
	}

	return namespace
}
