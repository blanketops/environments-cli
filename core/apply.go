package core

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ApplyRawYAML applies raw YAML text using robustApply
func ApplyRawYAML(dc dynamic.Interface, mapper meta.RESTMapper, data []byte) error {
	objs, err := decodeYAMLStream(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return robustApply(dc, mapper, objs)
}

// applyUnstructured performs create-or-update (safe for CRDs) using REST mapping.
func applyUnstructured(dc dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to map GVK %v: %w", gvk, err)
	}

	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = "default"
			obj.SetNamespace(ns)
		}
		ri = dc.Resource(mapping.Resource).Namespace(ns)
	} else {
		ri = dc.Resource(mapping.Resource)
	}

	// Try get
	existing, err := ri.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
	if err == nil {
		// update path: set resource version and update
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, uerr := ri.Update(context.Background(), obj, metav1.UpdateOptions{})
		return uerr
	}

	// create path
	_, cerr := ri.Create(context.Background(), obj, metav1.CreateOptions{})
	return cerr
}

// createUnstructured attempts a straight create (fail if exists)
func createUnstructured(dc dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to map GVK %v: %w", gvk, err)
	}

	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = "default"
			obj.SetNamespace(ns)
		}
		ri = dc.Resource(mapping.Resource).Namespace(ns)
	} else {
		ri = dc.Resource(mapping.Resource)
	}

	_, cerr := ri.Create(context.Background(), obj, metav1.CreateOptions{})
	return cerr
}

// deleteUnstructured deletes object by name using mapper.
func deleteUnstructured(dc dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to map GVK %v: %w", gvk, err)
	}

	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = "default"
			obj.SetNamespace(ns)
		}
		ri = dc.Resource(mapping.Resource).Namespace(ns)
	} else {
		ri = dc.Resource(mapping.Resource)
	}

	deletePolicy := metav1.DeletePropagationForeground
	return ri.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
}

// robustApply applies a list of objects with kubectl-like logic:
//
//	✔ CRDs first
//	✔ retry discovery mapping (CRDs may not be ready instantly)
//	✔ correct cluster-scoped handling (NO Namespace())
//	✔ create-or-update
func robustApply(dc dynamic.Interface, mapper meta.RESTMapper, objs []*unstructured.Unstructured) error {

	fmt.Println("📌 Preparing resources...")

	// 1. Split CRDs first
	var crds, others []*unstructured.Unstructured
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		if strings.ToLower(gvk.Group) == "apiextensions.k8s.io" && gvk.Kind == "CustomResourceDefinition" {
			crds = append(crds, o)
		} else {
			others = append(others, o)
		}
	}

	// Helper to apply a single object
	applyOne := func(o *unstructured.Unstructured) error {
		gvk := o.GroupVersionKind()
		fmt.Printf("🔧 Applying %s %s\n", gvk.Kind, o.GetName())

		// Get REST mapping with retry
		var mapping *meta.RESTMapping
		var err error

		backoff := wait.Backoff{Steps: 6, Duration: 400 * time.Millisecond, Factor: 1.3}
		err = wait.ExponentialBackoff(backoff, func() (bool, error) {
			mapping, err = mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				if meta.IsNoMatchError(err) ||
					strings.Contains(err.Error(), "no matches for kind") {
					return false, nil
				}
				return false, err
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("mapping failed for %v: %w", gvk, err)
		}

		var ri dynamic.ResourceInterface

		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := o.GetNamespace()
			if ns == "" {
				ns = "default"
				o.SetNamespace(ns)
			}
			ri = dc.Resource(mapping.Resource).Namespace(ns)
		} else {
			// cluster-scoped MUST NOT have namespace
			o.SetNamespace("")
			ri = dc.Resource(mapping.Resource)
		}

		// Try GET → Update OR fallback Create
		existing, err := ri.Get(context.Background(), o.GetName(), metav1.GetOptions{})
		if err == nil {
			// update path
			o.SetResourceVersion(existing.GetResourceVersion())
			_, err := ri.Update(context.Background(), o, metav1.UpdateOptions{})
			if err == nil {
				//fmt.Printf("   ✔ Updated %s %s\n", gvk.Kind, o.GetName())
			}
			return err
		}

		// create path
		_, err = ri.Create(context.Background(), o, metav1.CreateOptions{})
		if err == nil {
			//fmt.Printf("   ✔ Created %s %s\n", gvk.Kind, o.GetName())
		}
		return err
	}

	// 2. Apply CRDs FIRST
	if len(crds) > 0 {
		fmt.Println("📘 Applying CRDs first...")
		for _, c := range crds {
			if err := applyOne(c); err != nil {
				return fmt.Errorf("CRD apply failed: %w", err)
			}
		}
		fmt.Println("⏳ Waiting for CRDs to register in discovery...")
		time.Sleep(1 * time.Second)
	}

	// 3. Apply everything else
	fmt.Println("📦 Applying remaining resources...")
	for _, o := range others {
		if err := applyOne(o); err != nil {
			return err
		}
	}

	return nil
}
