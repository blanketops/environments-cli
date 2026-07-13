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
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/blanketops/environments-cli/util"
)

// operatorInstallerURL points at the consolidated installer YAML the
// environments-install repo publishes on every tagged release — CRDs,
// RBAC, and the controller-manager Deployment, built by that repo's
// `make build-installer` (kustomize build config/default) and uploaded as
// "install.yaml" (see its release.yml). GitHub's releases/latest/download
// redirect always resolves to the current release's copy of that asset.
const operatorInstallerURL = "https://github.com/blanketops/environments-install/releases/latest/download/install.yaml"

// InstallOperator applies the latest published operator bundle (CRDs,
// RBAC, controller-manager Deployment) to the current kube context. It's
// the runtime counterpart to SelfInstall, which only fetches this CLI's
// own binary — the CLI and the operator it drives are versioned and
// released independently.
func InstallOperator() error {
	fmt.Println("📦 Installing BlanketOps Environments operator...")
	return ApplyFromURL(operatorInstallerURL)
}

// UninstallOperator removes the operator bundle applied by InstallOperator.
// It re-fetches the same installer YAML and deletes each object it
// declares, mirroring UninstallManifests but sourced from the release URL
// instead of an embedded path.
func UninstallOperator() error {
	fmt.Println("🗑️  Uninstalling BlanketOps Environments operator...")

	resp, err := http.Get(operatorInstallerURL)
	if err != nil {
		return fmt.Errorf("fetch operator installer: %w", err)
	}
	defer resp.Body.Close()

	objs, err := decodeYAMLDocuments(resp.Body)
	if err != nil {
		return fmt.Errorf("decode operator installer: %w", err)
	}

	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}
	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	p := util.NewSpinner(operatorInstallerURL)
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		p.Update(fmt.Sprintf("%s %s", gvk.Kind, o.GetName()))

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
	p.Done("")
	return nil
}
