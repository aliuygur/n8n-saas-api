//go:build ignore

package gke

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/aliuygur/n8n-saas-api/internal/kubeapply"
	"github.com/aliuygur/n8n-saas-api/internal/n8ntemplates"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps GKE and Kubernetes clients for deploying N8N instances
type Client struct {
	containerClient *container.ClusterManagerClient
	k8sClient       *kubernetes.Clientset
	restConfig      *rest.Config
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
	c.restConfig = config // Store the rest config for kubeapply
	return nil
}

func (c *Client) DeployN8NInstance(ctx context.Context, instance N8NInstance) error {
	if c.restConfig == nil {
		return fmt.Errorf("kubernetes client not connected")
	}

	// Map gke.N8NInstance to n8ntemplates.Config
	templateConfig := n8ntemplates.Config{
		Namespace:     instance.Namespace,
		EncryptionKey: instance.EncryptionKey,
		BaseURL:       instance.BaseURL,
		CPURequest:    instance.CPURequest,
		MemoryRequest: instance.MemoryRequest,
		CPULimit:      instance.CPULimit,
		MemoryLimit:   instance.MemoryLimit,
		StorageSize:   instance.StorageSize,
	}

	// Render templates using n8ntemplates package
	yaml, err := n8ntemplates.Render(templateConfig)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Apply YAML using kubeapply package
	if err := kubeapply.ApplyYAMLString(ctx, c.restConfig, yaml); err != nil {
		return fmt.Errorf("failed to apply YAML: %w", err)
	}

	return nil
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
