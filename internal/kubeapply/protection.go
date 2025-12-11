package kubeapply

import (
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ProtectionConfig controls blacklist/whitelist for resources
type ProtectionConfig struct {
	// entries are strings like "group/version/Kind" or "v1/ConfigMap" for core group
	Blacklist []string
	Whitelist []string
}

// DefaultProtection returns a sane default protection config blocking CRDs and cluster RBAC/webhooks
func DefaultProtection() ProtectionConfig {
	return ProtectionConfig{
		Blacklist: []string{
			"apiextensions.k8s.io/v1/CustomResourceDefinition",
			"rbac.authorization.k8s.io/v1/ClusterRole",
			"rbac.authorization.k8s.io/v1/ClusterRoleBinding",
			"admissionregistration.k8s.io/v1/ValidatingWebhookConfiguration",
			"admissionregistration.k8s.io/v1/MutatingWebhookConfiguration",
		},
		Whitelist: []string{
			"apps/v1/Deployment",
			"v1/ConfigMap",
			"v1/Service",
		},
	}
}

// isAllowed returns true if operations on the given GVK are allowed per protection config.
// Policy: if in blacklist -> deny. Else if in whitelist -> allow. Else allow by default.
func (p ProtectionConfig) isAllowed(gvk schema.GroupVersionKind) bool {
	key := fmt.Sprintf("%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	key = strings.TrimPrefix(key, "/") // in case Group is empty -> "/v1/Pod", trim to "v1/Pod"

	// blacklist has priority
	if slices.Contains(p.Blacklist, key) {
		return false
	}

	// whitelist explicit allow
	if slices.Contains(p.Whitelist, key) {
		return true
	}

	// default allow
	return true
}
