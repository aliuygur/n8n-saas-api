package provisioning

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/db"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Logs API
type LogsRequest struct {
	InstanceID int    `json:"instance_id"`
	Component  string `json:"component"` // main, worker, postgres, redis
	Lines      int    `json:"lines,omitempty"`
	Follow     bool   `json:"follow,omitempty"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Pod       string    `json:"pod"`
	Container string    `json:"container"`
	Message   string    `json:"message"`
}

type LogsResponse struct {
	Logs []*LogEntry `json:"logs"`
}

//encore:api private
func (s *Service) GetLogs(ctx context.Context, req *LogsRequest) (*LogsResponse, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return nil, fmt.Errorf("instance must be deployed to view logs")
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Determine pod selector based on component
	labelSelector := ""
	switch req.Component {
	case "main":
		labelSelector = "app=n8n-main,component=main"
	case "worker":
		labelSelector = "app=n8n-worker,component=worker"
	case "postgres":
		labelSelector = "app=postgres"
	case "redis":
		labelSelector = "app=redis"
	default:
		labelSelector = "app in (n8n-main,n8n-worker,postgres,redis)"
	}

	logs, err := s.getPodLogs(ctx, instance.Namespace, labelSelector, req.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return &LogsResponse{Logs: logs}, nil
}

// Metrics API
type MetricsRequest struct {
	InstanceID int `json:"instance_id"`
}

type MetricsResponse struct {
	InstanceID int       `json:"instance_id"`
	Timestamp  time.Time `json:"timestamp"`
	Metrics    string    `json:"metrics"` // JSON string instead of interface{}
}

//encore:api private
func (s *Service) GetMetrics(ctx context.Context, req *MetricsRequest) (*MetricsResponse, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return nil, fmt.Errorf("instance must be deployed to view metrics")
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// TODO: Implement GetInstanceStatus method for SQLite-based architecture
	// For now, return basic status information
	basicMetrics := map[string]interface{}{
		"namespace":    instance.Namespace,
		"status":       "running", // Assume running for now
		"architecture": "sqlite_isolated",
	}

	metricsJSON, err := json.Marshal(basicMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return &MetricsResponse{
		InstanceID: int(instance.ID),
		Timestamp:  time.Now(),
		Metrics:    string(metricsJSON),
	}, nil
}

// Health API
type HealthRequest struct {
	InstanceID int `json:"instance_id"`
}

type ComponentHealth struct {
	Status      string    `json:"status"` // healthy, degraded, unhealthy
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

type HealthStatus struct {
	InstanceID  int                        `json:"instance_id"`
	Overall     string                     `json:"overall"` // healthy, degraded, unhealthy
	Components  map[string]ComponentHealth `json:"components"`
	LastChecked time.Time                  `json:"last_checked"`
}

//encore:api private
func (s *Service) GetHealth(ctx context.Context, req *HealthRequest) (*HealthStatus, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return &HealthStatus{
			InstanceID:  int(instance.ID),
			Overall:     "unhealthy",
			Components:  map[string]ComponentHealth{},
			LastChecked: time.Now(),
		}, nil
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	health, err := s.checkInstanceHealth(ctx, instance.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}

	health.InstanceID = int(instance.ID)
	return health, nil
}

// Resource Usage API
type ResourceUsageRequest struct {
	InstanceID int `json:"instance_id"`
}

type ResourceUsage struct {
	InstanceID int       `json:"instance_id"`
	Timestamp  time.Time `json:"timestamp"`
	CPU        string    `json:"cpu"`     // JSON string instead of interface{}
	Memory     string    `json:"memory"`  // JSON string instead of interface{}
	Storage    string    `json:"storage"` // JSON string instead of interface{}
}

//encore:api private
func (s *Service) GetResourceUsage(ctx context.Context, req *ResourceUsageRequest) (*ResourceUsage, error) {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return nil, fmt.Errorf("instance must be deployed to view resource usage")
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	usage, err := s.getResourceUsage(ctx, instance.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource usage: %w", err)
	}

	usage.InstanceID = int(instance.ID)
	return usage, nil
}

// Restart Component API
type RestartComponentRequest struct {
	InstanceID int    `json:"instance_id"`
	Component  string `json:"component"` // main, worker, postgres, redis
}

//encore:api private
func (s *Service) RestartComponent(ctx context.Context, req *RestartComponentRequest) error {
	queries := db.New(s.db)

	instance, err := queries.GetInstance(ctx, int32(req.InstanceID))
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status != "deployed" {
		return fmt.Errorf("instance must be deployed to restart components")
	}

	// Connect to GKE cluster
	if err := s.gke.ConnectToCluster(ctx, s.config.DefaultClusterName, s.config.DefaultZone); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Restart component
	if err := s.restartComponent(ctx, instance, req.Component); err != nil {
		return fmt.Errorf("failed to restart component: %w", err)
	}

	return nil
}

func (s *Service) getPodLogs(ctx context.Context, namespace, labelSelector string, lines int) ([]*LogEntry, error) {
	k8sClient := s.gke.K8sClient()

	if lines == 0 {
		lines = 100
	}

	// Get pods matching the selector
	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var allLogs []*LogEntry

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			logOptions := &corev1.PodLogOptions{
				Container: container.Name,
				TailLines: func(i int) *int64 { i64 := int64(i); return &i64 }(lines),
			}

			req := k8sClient.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions)
			logs, err := req.Stream(ctx)
			if err != nil {
				log.Printf("Failed to get logs for pod %s container %s: %v", pod.Name, container.Name, err)
				continue
			}

			logData, err := io.ReadAll(logs)
			logs.Close()
			if err != nil {
				log.Printf("Failed to read logs for pod %s: %v", pod.Name, err)
				continue
			}

			// Parse log lines
			logLines := strings.Split(string(logData), "\n")
			for _, line := range logLines {
				if strings.TrimSpace(line) == "" {
					continue
				}

				allLogs = append(allLogs, &LogEntry{
					Timestamp: time.Now(), // In real implementation, parse timestamp from log line
					Pod:       pod.Name,
					Container: container.Name,
					Message:   line,
				})
			}
		}
	}

	return allLogs, nil
}

func (s *Service) checkInstanceHealth(ctx context.Context, namespace string) (*HealthStatus, error) {
	k8sClient := s.gke.K8sClient()

	components := map[string]ComponentHealth{}
	overall := "healthy"

	// Check deployments
	deployments, err := k8sClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, dep := range deployments.Items {
		status := "healthy"
		message := "All replicas ready"

		if dep.Status.ReadyReplicas < *dep.Spec.Replicas {
			status = "degraded"
			message = fmt.Sprintf("%d/%d replicas ready", dep.Status.ReadyReplicas, *dep.Spec.Replicas)
			if overall == "healthy" {
				overall = "degraded"
			}
		}

		if dep.Status.ReadyReplicas == 0 {
			status = "unhealthy"
			message = "No replicas ready"
			overall = "unhealthy"
		}

		components[dep.Name] = ComponentHealth{
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
		}
	}

	// Check services
	services, err := k8sClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, svc := range services.Items {
		components[fmt.Sprintf("service-%s", svc.Name)] = ComponentHealth{
			Status:      "healthy",
			Message:     "Service active",
			LastChecked: time.Now(),
		}
	}

	return &HealthStatus{
		Overall:     overall,
		Components:  components,
		LastChecked: time.Now(),
	}, nil
}

func (s *Service) getResourceUsage(ctx context.Context, namespace string) (*ResourceUsage, error) {
	k8sClient := s.gke.K8sClient()

	// Get pods to check resource usage
	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Initialize counters
	cpuRequests := 0
	cpuLimits := 0
	memoryRequests := int64(0)
	memoryLimits := int64(0)

	// Note: In a real implementation, you would use metrics-server or Prometheus
	// to get actual resource usage. This is a simplified version.

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests.Cpu(); cpu != nil {
					cpuRequests += int(cpu.MilliValue())
				}
				if memory := container.Resources.Requests.Memory(); memory != nil {
					memoryRequests += memory.Value()
				}
			}
			if container.Resources.Limits != nil {
				if cpu := container.Resources.Limits.Cpu(); cpu != nil {
					cpuLimits += int(cpu.MilliValue())
				}
				if memory := container.Resources.Limits.Memory(); memory != nil {
					memoryLimits += memory.Value()
				}
			}
		}
	}

	// Create JSON strings for each resource type
	cpuJSON, _ := json.Marshal(map[string]int{
		"requests": cpuRequests,
		"limits":   cpuLimits,
	})

	memoryJSON, _ := json.Marshal(map[string]int64{
		"requests": memoryRequests,
		"limits":   memoryLimits,
	})

	storageJSON, _ := json.Marshal(map[string]int{
		"used":  0, // Placeholder
		"total": 0, // Placeholder
	})

	usage := &ResourceUsage{
		Timestamp: time.Now(),
		CPU:       string(cpuJSON),
		Memory:    string(memoryJSON),
		Storage:   string(storageJSON),
	}

	return usage, nil
}

func (s *Service) restartComponent(ctx context.Context, instance db.Instance, component string) error {
	k8sClient := s.gke.K8sClient()

	log.Printf("Starting component restart: instance_id=%d component=%s", instance.ID, component)

	var deploymentName string
	switch component {
	case "main":
		deploymentName = "n8n-main"
	case "worker":
		deploymentName = "n8n-worker"
	case "postgres":
		deploymentName = "postgres"
	case "redis":
		deploymentName = "redis"
	default:
		return fmt.Errorf("invalid component name: %s", component)
	}

	// Get deployment
	deployment, err := k8sClient.AppsV1().Deployments(instance.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Restart by updating an annotation
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = k8sClient.AppsV1().Deployments(instance.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	log.Printf("Component restart completed: instance_id=%d component=%s", instance.ID, component)
	return nil
}
