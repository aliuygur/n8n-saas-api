package gke

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/samber/lo"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps GKE and Kubernetes clients for deploying N8N instances
type Client struct {
	containerClient *container.ClusterManagerClient
	k8sClient       *kubernetes.Clientset
	projectID       string
	credentialsJSON []byte
}

// N8NInstance represents an N8N deployment configuration
type N8NInstance struct {
	Namespace     string
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
	StorageSize   string
	EncryptionKey string

	BaseURL string
}

// Constants for N8N component names
const (
	N8NMainName = "n8n-main"
	N8NDBName   = "n8n-db"
)

func NewClient(projectID string, credentialsJSON []byte) (*Client, error) {
	ctx := context.Background()

	containerClient, err := container.NewClusterManagerClient(ctx, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create container client: %w", err)
	}

	// ping the client to verify connection
	_, err = containerClient.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ping GKE API: %w", err)
	}

	return &Client{
		containerClient: containerClient,
		projectID:       projectID,
		credentialsJSON: credentialsJSON,
	}, nil
}

// Add getter method to expose k8s client
func (c *Client) K8sClient() kubernetes.Interface {
	return c.k8sClient
}

func (c *Client) ConnectToCluster(ctx context.Context, clusterName, zone string) error {
	cluster, err := c.containerClient.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", c.projectID, zone, clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Decode the base64-encoded certificate
	caData, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return fmt.Errorf("failed to decode cluster CA certificate: %w", err)
	}

	// Get credentials from the stored credentialsJSON
	creds, err := google.CredentialsFromJSON(ctx, c.credentialsJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("failed to create credentials from JSON: %w", err)
	}

	// Create the rest config
	config := &rest.Config{
		Host: "https://" + cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
	}

	// Create token source
	token, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	config.BearerToken = token.AccessToken

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	c.k8sClient = clientset
	return nil
}

func (c *Client) DeployN8NInstance(ctx context.Context, instance N8NInstance) error {
	if c.k8sClient == nil {
		return fmt.Errorf("kubernetes client not connected")
	}

	// Create namespace
	if err := c.createNamespace(ctx, instance.Namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create PVC for SQLite database
	if err := c.createPVC(ctx, instance); err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	// Deploy N8N instance with SQLite
	if err := c.deployN8N(ctx, instance); err != nil {
		return fmt.Errorf("failed to deploy N8N: %w", err)
	}

	// Create service
	if err := c.createService(ctx, instance); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Note: No ingress or certificates needed since we use Cloudflare tunnel for external access

	return nil
}

func (c *Client) createNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"name": namespace,
			},
		},
	}

	_, err := c.k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	// Ignore error if namespace already exists
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}
func (c *Client) createPVC(ctx context.Context, instance N8NInstance) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      N8NMainName + "-data",
			Namespace: instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(instance.StorageSize),
				},
			},
			StorageClassName: lo.ToPtr("premium-rwo"),
		},
	}

	_, err := c.k8sClient.CoreV1().PersistentVolumeClaims(instance.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	return err
}

func (c *Client) deployN8N(ctx context.Context, instance N8NInstance) error {
	// Build pod annotations
	annotations := map[string]string{
		"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
		"run.googleapis.com/cpu-throttling":              "false",
		"run.googleapis.com/execution-environment":       "gen2",
	}

	// Build pod spec
	podSpec := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: lo.ToPtr(int64(1000)),
		},
		Containers: []corev1.Container{
			{
				Name:  "n8n",
				Image: "n8nio/n8n:latest",
				Env: []corev1.EnvVar{
					{Name: "N8N_USER_FOLDER", Value: "/data"},
					{Name: "N8N_ENCRYPTION_KEY", Value: instance.EncryptionKey},
					{Name: "GENERIC_TIMEZONE", Value: "UTC"},
					{Name: "NODE_ENV", Value: "production"},
					{Name: "N8N_EDITOR_BASE_URL", Value: instance.BaseURL},

					// Database
					{Name: "DB_TYPE", Value: "sqlite"},
					{Name: "DB_SQLITE_DATABASE", Value: "/data/database.sqlite"},
					{Name: "DB_SQLITE_VACUUM_ON_STARTUP", Value: "true"},

					// Executions
					{Name: "EXECUTIONS_DATA_MAX_AGE", Value: "168"}, // 7 days

					// Logs
					{Name: "N8N_LOG_LEVEL", Value: "warn"},

					// Nodes
					{Name: "NODE_FUNCTION_ALLOW_BUILTIN", Value: "*"},
				},
				Ports: []corev1.ContainerPort{
					{ContainerPort: 5678},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "n8n-data",
						MountPath: "/data",
					},
				},
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:  lo.ToPtr(int64(1000)), // Run as node user
					RunAsGroup: lo.ToPtr(int64(1000)),
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(instance.CPURequest),
						corev1.ResourceMemory: resource.MustParse(instance.MemoryRequest),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(instance.CPULimit),
						corev1.ResourceMemory: resource.MustParse(instance.MemoryLimit),
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "n8n-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: N8NMainName + "-data",
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      N8NMainName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: lo.ToPtr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": N8NMainName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": N8NMainName,
					},
					Annotations: annotations,
				},
				Spec: podSpec,
			},
		},
	}

	_, err := c.k8sClient.AppsV1().Deployments(instance.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	return err
}

func (c *Client) createService(ctx context.Context, instance N8NInstance) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      N8NMainName,
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": N8NMainName,
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(5678),
				},
			},
		},
	}

	_, err := c.k8sClient.CoreV1().Services(instance.Namespace).Create(ctx, service, metav1.CreateOptions{})
	return err
}

// NamespaceExists checks if a namespace exists in the cluster
func (c *Client) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	if c.k8sClient == nil {
		return false, fmt.Errorf("kubernetes client not connected")
	}

	_, err := c.k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteN8NInstance deletes an N8N instance and its associated resources
func (c *Client) DeleteN8NInstance(ctx context.Context, namespace string) error {
	if c.k8sClient == nil {
		return fmt.Errorf("kubernetes client not connected")
	}

	// Delete the entire namespace (this will delete all resources within it)
	err := c.k8sClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to delete namespace %s: %w", namespace, err)
	}

	return nil
}
