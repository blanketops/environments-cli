/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

// ----------------------
// Wait helpers (used earlier in main.go)
// ----------------------

// WaitForDaemonSetReady waits until a daemonset has the desired number of ready pods.
func WaitForDaemonSetReady(namespace, name string) error {
	dc, _, err := NewDynamicClient()
	if err != nil {
		return err
	}
	// Use apps/v1 DaemonSet via dynamic client by mapping the resource.
	mapper, err := NewRESTMapperPointerFor(dc)
	if err != nil {
		return err
	}

	return wait.PollImmediate(3*time.Second, 4*time.Minute, func() (bool, error) {
		// Build a fake Unstructured representing the DaemonSet GVK to get mapping.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"})
		mapping, err := mapper.RESTMapping(u.GroupVersionKind().GroupKind(), u.GroupVersionKind().Version)
		if err != nil {
			// discovery may not yet be ready; retry
			return false, nil
		}
		var ri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ri = dc.Resource(mapping.Resource).Namespace(namespace)
		} else {
			ri = dc.Resource(mapping.Resource)
		}
		dsObj, err := ri.Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		status := dsObj.Object["status"].(map[string]any)
		desired, _ := status["desiredNumberScheduled"].(int64)
		ready, _ := status["numberReady"].(int64)
		// fallback: if types mismatch, try numeric conversions
		if desired == ready {
			return true, nil
		}
		return false, nil
	})
}

// WaitForDeploymentReady waits until the named deployment reports all replicas ready.
func WaitForDeploymentReady(namespace, name string) error {
	dc, _, err := NewDynamicClient()
	if err != nil {
		return err
	}
	mapper, err := NewRESTMapperPointerFor(dc)
	if err != nil {
		return err
	}

	return wait.PollImmediate(3*time.Second, 4*time.Minute, func() (bool, error) {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
		mapping, err := mapper.RESTMapping(u.GroupVersionKind().GroupKind(), u.GroupVersionKind().Version)
		if err != nil {
			return false, nil
		}
		var ri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ri = dc.Resource(mapping.Resource).Namespace(namespace)
		} else {
			ri = dc.Resource(mapping.Resource)
		}
		depObj, err := ri.Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		status, _ := depObj.Object["status"].(map[string]any)
		readyReplicas, _ := status["readyReplicas"].(int64)
		specReplicas, _ := depObj.Object["spec"].(map[string]any)
		var desired int64 = 0
		if rr, ok := specReplicas["replicas"].(int64); ok {
			desired = rr
		}
		if desired == 0 {
			// fallback: assume desired = readyReplicas when not set explicitly
			desired = readyReplicas
		}
		if readyReplicas >= desired {
			return true, nil
		}
		return false, nil
	})
}

// WaitForAllNodesReady waits until all nodes are Ready.
func WaitForAllNodesReady() error {
	dc, _, err := NewDynamicClient()
	if err != nil {
		return err
	}
	mapper, err := NewRESTMapperPointerFor(dc)
	if err != nil {
		return err
	}
	// Use core/v1 Node resource
	return wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"})
		mapping, err := mapper.RESTMapping(u.GroupVersionKind().GroupKind(), u.GroupVersionKind().Version)
		if err != nil {
			return false, nil
		}
		var ri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ri = dc.Resource(mapping.Resource).Namespace("") // nodes are cluster scoped
		} else {
			ri = dc.Resource(mapping.Resource)
		}
		listObj, err := ri.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		items, _ := listObj.Object["items"].([]any)
		for _, it := range items {
			// each item is map[string]any
			m, ok := it.(map[string]any)
			if !ok {
				return false, nil
			}
			status, _ := m["status"].(map[string]any)
			conds, _ := status["conditions"].([]any)
			readyFound := false
			for _, c := range conds {
				cm, _ := c.(map[string]any)
				if t, ok := cm["type"].(string); ok && t == "Ready" {
					if s, ok := cm["status"].(string); ok && s == "True" {
						readyFound = true
						break
					}
				}
			}
			if !readyFound {
				return false, nil
			}
		}
		return true, nil
	})
}
