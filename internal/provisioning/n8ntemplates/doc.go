// Package n8ntemplates provides N8N Kubernetes template management and rendering.
//
// This package abstracts away the template management and rendering logic for
// deploying N8N instances to Kubernetes. It includes embedded YAML templates
// for namespace, PVC, deployment, and service resources.
//
// # Features
//
// - Embedded templates using go:embed (no external files needed)
// - Simple configuration with placeholder replacement
// - Pure in-memory rendering (no filesystem access required)
// - Single multi-document YAML output
//
// # Usage
//
// Render all templates to a single multi-document YAML:
//
//	config := n8ntemplates.Config{
//	    Namespace:     "n8n-user-abc123",
//	    EncryptionKey: "secure-key-here",
//	    BaseURL:       "https://user.example.com",
//	    CPURequest:    "150m",
//	    MemoryRequest: "512Mi",
//	    CPULimit:      "500m",
//	    MemoryLimit:   "1Gi",
//	    StorageSize:   "5Gi",
//	}
//
//	yaml, err := n8ntemplates.Render(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// yaml now contains all resources separated by ---
//
// # Templates
//
// The package includes a single embedded template file (n8n.yaml) containing all resources:
//
//   - Namespace: Kubernetes Namespace definition
//   - PersistentVolumeClaim: N8N data storage (5Gi, premium-rwo)
//   - Deployment: N8N Deployment with container configuration
//   - Service: ClusterIP Service for internal access (port 80 → 5678)
//
// All templates use PLACEHOLDER_ prefixed variables that are replaced at render time.
// No external tools (like Kustomize) are required.
package n8ntemplates
