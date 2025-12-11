package kubeapply

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig tries in-cluster first, then falls back to local kubeconfig (KUBECONFIG/$HOME/.kube/config)
func GetConfig() (*rest.Config, error) {
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
