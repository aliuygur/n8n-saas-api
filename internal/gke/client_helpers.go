package gke

import "k8s.io/client-go/kubernetes"

// Add getter method to expose k8s client
func (c *Client) K8sClient() kubernetes.Interface {
	return c.k8sClient
}
