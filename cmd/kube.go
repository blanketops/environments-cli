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
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// ----------------------
// Kube clients & mapper
// ----------------------

// BuildConfig loads kubeconfig in order: KUBECONFIG, $HOME/.kube/config, in-cluster.
func BuildConfig() (*rest.Config, error) {
	if kube := os.Getenv("KUBECONFIG"); kube != "" {
		if cfg, err := clientcmd.BuildConfigFromFlags("", kube); err == nil {
			return cfg, nil
		}
		// fall through on error
	}

	if home, err := os.UserHomeDir(); err == nil {
		kf := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kf); err == nil {
			if cfg, err := clientcmd.BuildConfigFromFlags("", kf); err == nil {
				return cfg, nil
			}
		}
	}

	// in-cluster
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("no kubeconfig found and in-cluster config failed: %w", err)
	}
	return cfg, nil
}

// NewDynamicClient returns a dynamic client and associated rest.Config.
func NewDynamicClient() (dynamic.Interface, *rest.Config, error) {
	cfg, err := BuildConfig()
	if err != nil {
		return nil, nil, err
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return dc, cfg, nil
}

// NewRESTMapper builds a discovery-backed RESTMapper using the provided rest.Config.
func NewRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed creating discovery client: %w", err)
	}
	apiGroupResources, err := restmapper.GetAPIGroupResources(disco)
	if err != nil {
		return nil, fmt.Errorf("failed getting API group resources: %w", err)
	}
	return restmapper.NewDiscoveryRESTMapper(apiGroupResources), nil
}

// ----------------------
// Mapper pointer helper (caching small helper)
// ----------------------

// NewRESTMapperPointerFor builds a restmapper backed by discovery; returned mapper is meta.RESTMapper.
func NewRESTMapperPointerFor(dc dynamic.Interface) (meta.RESTMapper, error) {
	// To build the RESTMapper we need the rest.Config — dynamic client doesn't surface it,
	// so rebuild config via BuildConfig() (cheap) and call NewRESTMapper.
	cfg, err := BuildConfig()
	if err != nil {
		return nil, err
	}
	return NewRESTMapper(cfg)
}
