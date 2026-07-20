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

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Dependency describes one individually addressable piece of the platform
// stack: its embedded manifests (if any), how to probe its install status,
// and — where "apply the manifests" or "delete the manifests" isn't the
// whole story (Helm-installed components, webhook cert setup, remote CRD
// fetches) — install/uninstall overrides.
//
// Status is probed one of two ways: most components install into their own
// namespace, so Namespace is enough (existence of the namespace is a coarse
// but cheap installed/not-installed signal). Components with no namespace
// of their own — e.g. buildstrategies, which only creates cluster-scoped
// ClusterBuildStrategy objects — set StatusGVR/StatusName instead, probing
// a specific cluster-scoped resource directly rather than falling back to
// "unknown".
//
// This registry exists so each component can be created/read/updated/
// deleted on its own via `bops-env dependencies <verb> <name>`, alongside
// the existing all-at-once InstallAll/UninstallAll in all.go. It does not
// encode install ORDER between dependencies — that dependency-aware
// sequence is explicit in InstallAll — installing a single dependency is
// the caller's responsibility to sequence correctly (e.g. crossplane's
// provider needs crossplane itself already running).
type Dependency struct {
	Name        string
	Description string
	Manifests   []string
	Namespace   string // namespaced status probe; empty if StatusGVR is set instead
	HelmRelease string // set only for Helm-installed components, used for status/uninstall

	// StatusGVR/StatusName probe a specific cluster-scoped resource for
	// components with no namespace of their own. Both must be set together.
	StatusGVR  schema.GroupVersionResource
	StatusName string

	install   func() error
	uninstall func() error
}

// registry is the addressable set of dependencies. calico and multus are
// intentionally absent — see the comment on InstallAll in all.go.
var registry = []Dependency{
	{
		Name:        "carvel",
		Description: "Carvel Kapp Controller — packaging and lifecycle management",
		Manifests:   []string{"dependencies/carvel/release.yaml"},
		Namespace:   "kapp-controller",
	},
	{
		Name:        "argoevents",
		Description: "Argo Events — event-driven pipelines",
		Manifests:   []string{"dependencies/argoevents/manifest.yaml"},
		Namespace:   "argo-events",
	},
	{
		Name:        "tekton-pipelines",
		Description: "Tekton Pipelines — CI/CD execution engine",
		Manifests:   []string{"dependencies/tekton/tekton_pipelines.yaml"},
		Namespace:   "tekton-pipelines",
	},
	{
		Name:        "tekton-dashboard",
		Description: "Tekton Dashboard — pipeline UI",
		Manifests:   []string{"dependencies/tekton/tekton_dashboard.yaml"},
		Namespace:   "tekton-pipelines",
	},
	{
		Name:        "shipwright",
		Description: "Shipwright Build — Kubernetes-native image builds",
		Manifests:   []string{"dependencies/shipwright/shipwright_build.yaml"},
		Namespace:   "shipwright-build",
		install: func() error {
			if err := DependenciesInstall([]string{"dependencies/shipwright/shipwright_build.yaml"}); err != nil {
				return err
			}
			// The webhook doesn't work without its TLS certs generated,
			// approved, and the deployment restarted to pick up the CA
			// bundle — not optional, see all.go's InstallAll.
			return RunShipwrightCertSetup()
		},
	},
	{
		Name:        "crossplane",
		Description: "Crossplane — infrastructure orchestration",
		Manifests:   []string{"dependencies/crossplane/provider.yaml"},
		Namespace:   "crossplane-system",
		HelmRelease: "crossplane",
		install: func() error {
			if err := InstallCrossplane(); err != nil {
				return err
			}
			// Needs Crossplane core (just installed above) already
			// running — its CRD is what this Provider resource uses.
			return DependenciesInstall([]string{"dependencies/crossplane/provider.yaml"})
		},
		uninstall: func() error {
			_ = helmUninstall("crossplane", "crossplane-system")
			return UninstallManifests([]string{"dependencies/crossplane/provider.yaml"})
		},
	},
	{
		Name:        "externalsecrets",
		Description: "External Secrets Operator — secure secret integration",
		Namespace:   "external-secrets",
		HelmRelease: "external-secrets",
		install:     InstallExternalSecrets,
		uninstall: func() error {
			return helmUninstall("external-secrets", "external-secrets")
		},
	},
	{
		Name:        "buildstrategies",
		Description: "Shipwright build strategies (buildpacks, kaniko, buildah)",
		Manifests: []string{
			"dependencies/cluster_strategies/buildpacks_v3.yaml",
			"dependencies/cluster_strategies/kaniko.yaml",
			"dependencies/cluster_strategies/buildah_shipwright_managed_push_cr.yaml",
		},
		// Cluster-scoped resources, no namespace of their own — probe one
		// of the three ClusterBuildStrategy objects directly instead.
		StatusGVR:  schema.GroupVersionResource{Group: "shipwright.io", Version: "v1beta1", Resource: "clusterbuildstrategies"},
		StatusName: "kaniko",
	},
	{
		Name:        "flux",
		Description: "Flux UI + config",
		Manifests:   []string{"dependencies/flux/fluxcdui.yaml"},
		Namespace:   "flux-system",
		HelmRelease: "flux",
		install: func() error {
			// Fetches Flux's CRDs before the manifest below needs them.
			if err := InstallFluxCRDs(); err != nil {
				return err
			}
			return DependenciesInstall([]string{"dependencies/flux/fluxcdui.yaml"})
		},
		uninstall: func() error {
			_ = helmUninstall("flux", "flux-system")
			return UninstallManifests([]string{"dependencies/flux/fluxcdui.yaml"})
		},
	},
	{
		Name:        "knative",
		Description: "Knative Serving — serverless workload runtime",
		Manifests: []string{
			"dependencies/knative/serving-crds.yaml",
			"dependencies/knative/serving-core.yaml",
		},
		Namespace: "knative-serving",
	},
	{
		Name:        "kourier",
		Description: "Kourier — Knative's ingress/networking layer",
		Manifests:   []string{"dependencies/kourier/kourier.yaml"},
		Namespace:   "kourier-system",
		install: func() error {
			if err := DependenciesInstall([]string{"dependencies/kourier/kourier.yaml"}); err != nil {
				return err
			}
			// Points Knative Serving's ingress-class at Kourier — needs
			// the Kourier controller (installed above) running first.
			return RunKnativeKourierSetup()
		},
	},
}

// FindDependency looks up a registered dependency by name.
func FindDependency(name string) (*Dependency, error) {
	for i := range registry {
		if registry[i].Name == name {
			return &registry[i], nil
		}
	}
	return nil, fmt.Errorf("unknown dependency %q (run 'bops-env dependencies list' to see available names)", name)
}

// DependencyNames returns the registered dependency names in registry order.
func DependencyNames() []string {
	names := make([]string, len(registry))
	for i, d := range registry {
		names[i] = d.Name
	}
	return names
}
