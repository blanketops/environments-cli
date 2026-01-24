package core

import (
	"bytes"
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// ---------------------------------------------------------------------------
// UninstallManifests
// ---------------------------------------------------------------------------
func UninstallManifests(manifestPaths []string) error {
	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	for _, path := range manifestPaths {
		fmt.Printf("❌ Removing %s\n", path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read manifest %s: %w", path, err)
		}

		objs, err := decodeYAMLStream(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}

		for _, o := range objs {
			gvk := o.GroupVersionKind()

			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				continue
			}

			var ri dynamic.ResourceInterface
			if mapping.Scope.Name() == "namespace" {
				ns := o.GetNamespace()
				if ns == "" {
					ns = "default"
				}
				ri = dc.Resource(mapping.Resource).Namespace(ns)
			} else {
				ri = dc.Resource(mapping.Resource)
			}

			_ = ri.Delete(context.Background(), o.GetName(), metav1.DeleteOptions{})
		}
	}

	return nil
}
