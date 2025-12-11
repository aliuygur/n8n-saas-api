package kubeapply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// ApplyYAML converts YAML → Unstructured and applies to cluster.
func ApplyYAML(ctx context.Context, cfg *rest.Config, yamlData []byte) error {
	obj, gvk, jsonData, err := yamlToUnstructured(yamlData)
	if err != nil {
		return fmt.Errorf("convert yaml: %w", err)
	}

	dc, rm, err := makeClients(cfg)
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
			FieldManager: "go-kube-apply",
			Force:        ptr(true),
		},
	)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	fmt.Printf("Applied: %s/%s\n", mapping.Resource.Resource, applied.GetName())
	return nil
}

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

func makeClients(cfg *rest.Config) (dynamic.Interface, meta.RESTMapper, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	cache := memory.NewMemCacheClient(disc)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return dc, mapper, nil
}

func ptr[T any](v T) *T { return &v }
