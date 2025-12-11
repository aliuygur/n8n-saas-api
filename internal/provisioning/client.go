package provisioning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Client handles N8N instance provisioning on Kubernetes
type Client struct {
	k8sClient  *kubernetes.Clientset
	restConfig *rest.Config
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
	BaseURL       string
}

// Constants for N8N component names
const (
	N8NMainName = "n8n-main"
	N8NDBName   = "n8n-db"
)

// NewClient creates a new provisioning client
// Tries in-cluster config first, then falls back to kubeconfig
func NewClient() (*Client, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		k8sClient:  clientset,
		restConfig: config,
	}, nil
}

// getConfig tries in-cluster first, then falls back to local kubeconfig (KUBECONFIG/$HOME/.kube/config)
func getConfig() (*rest.Config, error) {
	// in-cluster
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	// fallback to kubeconfig
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
}

// NewClientFromConfig creates a new provisioning client from rest.Config
func NewClientFromConfig(config *rest.Config) (*Client, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		k8sClient:  clientset,
		restConfig: config,
	}, nil
}

// K8sClient returns the Kubernetes clientset
func (c *Client) K8sClient() kubernetes.Interface {
	return c.k8sClient
}

// DeployN8NInstance deploys an N8N instance using YAML templates
func (c *Client) DeployN8NInstance(ctx context.Context, instance N8NInstance) error {
	if c.restConfig == nil {
		return fmt.Errorf("kubernetes client not connected")
	}

	// Map N8NInstance to n8ntemplates.Config
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
	renderedYAML, err := n8ntemplates.Render(templateConfig)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Apply YAML to cluster
	if err := c.applyMultiYAML(ctx, []byte(renderedYAML)); err != nil {
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

// applyMultiYAML applies multiple YAML documents separated by "---"
func (c *Client) applyMultiYAML(ctx context.Context, yamlData []byte) error {
	// Split by YAML document separator
	docs := splitYAMLDocuments(yamlData)

	// Apply each document
	for i, doc := range docs {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		if err := c.applyYAML(ctx, doc); err != nil {
			// Extract resource info for better error messages
			obj, gvk, _, parseErr := yamlToUnstructured(doc)
			if parseErr == nil {
				return fmt.Errorf("failed to apply document %d (%s/%s): %w", i+1, gvk.Kind, obj.GetName(), err)
			}
			return fmt.Errorf("failed to apply document %d: %w", i+1, err)
		}
	}

	return nil
}

// applyYAML converts YAML → Unstructured and applies to cluster using server-side apply
func (c *Client) applyYAML(ctx context.Context, yamlData []byte) error {
	obj, gvk, jsonData, err := yamlToUnstructured(yamlData)
	if err != nil {
		return fmt.Errorf("convert yaml: %w", err)
	}

	dc, rm, err := c.makeClients()
	if err != nil {
		return err
	}

	mapping, err := rm.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("rest mapping: %w", err)
	}

	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace("default")
		}
		ri = dc.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		ri = dc.Resource(mapping.Resource)
	}

	// PATCH (server-side apply)
	applied, err := ri.Patch(
		ctx,
		obj.GetName(),
		types.ApplyPatchType,
		jsonData,
		metav1.PatchOptions{
			FieldManager: "n8n-provisioning",
			Force:        ptr(true),
		},
	)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	fmt.Printf("Applied: %s/%s\n", mapping.Resource.Resource, applied.GetName())
	return nil
}

// makeClients creates dynamic client and REST mapper
func (c *Client) makeClients() (dynamic.Interface, meta.RESTMapper, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(c.restConfig)
	if err != nil {
		return nil, nil, err
	}

	cache := memory.NewMemCacheClient(disc)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	dc, err := dynamic.NewForConfig(c.restConfig)
	if err != nil {
		return nil, nil, err
	}

	return dc, mapper, nil
}

// splitYAMLDocuments splits multi-document YAML by "---" separator
func splitYAMLDocuments(data []byte) [][]byte {
	separator := []byte("\n---\n")
	docs := bytes.Split(data, separator)

	// Also handle "---" at the beginning of the file
	if len(docs) > 0 && len(bytes.TrimSpace(docs[0])) == 0 {
		docs = docs[1:]
	}

	return docs
}

// yamlToUnstructured converts YAML bytes to Unstructured object
func yamlToUnstructured(data []byte) (*unstructured.Unstructured, *schema.GroupVersionKind, []byte, error) {
	jsonData, err := yaml.ToJSON(data)
	if err != nil {
		return nil, nil, nil, err
	}

	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(jsonData); err != nil {
		return nil, nil, nil, err
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "" {
		return nil, nil, nil, errors.New("YAML missing Kind")
	}
	if gvk.Version == "" {
		return nil, nil, nil, errors.New("YAML missing apiVersion")
	}

	// Ensure jsonData is valid JSON in case Unstructured modified fields
	finalJSON, err := json.Marshal(u.Object)
	if err != nil {
		return nil, nil, nil, err
	}

	return u, &gvk, finalJSON, nil
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T { return &v }
