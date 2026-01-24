package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// ApplyManifest takes raw YAML bytes (possibly with multiple YAML docs)
// and applies them to the cluster using the dynamic client.
// This is fully kubectl-free and works in gokrazy static environments.
func ApplyManifest(dyn dynamic.Interface, data []byte) error {

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)

	for {
		// Each YAML doc → Unstructured object
		obj := &unstructured.Unstructured{}
		err := decoder.Decode(obj)

		if err != nil {
			if errors.Is(err, io.EOF) {
				break // done
			}
			return fmt.Errorf("failed to decode manifest: %w", err)
		}

		// Skip empty docs (e.g. "---\n")
		if len(obj.Object) == 0 {
			continue
		}

		// Convert GVK → GVR for dynamic client
		gvk := obj.GroupVersionKind()
		gvr, _ := meta.UnsafeGuessKindToResource(gvk)

		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}

		// Create the resource
		_, err = dyn.Resource(gvr).Namespace(namespace).Create(
			context.Background(),
			obj,
			metav1.CreateOptions{},
		)

		if err != nil {
			return fmt.Errorf(
				"failed to apply %s %s/%s: %w",
				gvk.Kind,
				namespace,
				obj.GetName(),
				err,
			)
		}
	}

	return nil
}
