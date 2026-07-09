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
)

// InstallAll installs the platform stack in an explicit, dependency-aware
// order. This used to be a generic walk over the embedded dependencies/
// tree, sorted alphabetically (with a couple of directory-priority hacks
// bolted on). That model was the root cause of a whole session's worth of
// ordering bugs: it has no way to express that Crossplane's provider.yaml
// needs Crossplane core (a Helm install, not an embedded manifest)
// running first, that cluster_strategies needs Shipwright's CRDs, or that
// Flux's manifest needs Flux's CRDs fetched first. This restores the
// original, explicit step-by-step sequence — each dependency knows what
// it needs before it, because it's written down, not inferred from
// directory names.
//
// calico and multus are intentionally absent: calico is currently
// breaking installs, and dependencies/multus/multus.yaml is (and always
// has been) an empty stub with nothing to apply.
func InstallAll() error {
	fmt.Println("📦 Installing Carvel Kapp...")
	if err := DependenciesInstall([]string{"dependencies/carvel/release.yaml"}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Argo Events...")
	if err := DependenciesInstall([]string{"dependencies/argoevents/manifest.yaml"}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Tekton Pipelines...")
	if err := DependenciesInstall([]string{"dependencies/tekton/tekton_pipelines.yaml"}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Tekton Dashboard...")
	if err := DependenciesInstall([]string{"dependencies/tekton/tekton_dashboard.yaml"}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Shipwright Build...")
	if err := DependenciesInstall([]string{"dependencies/shipwright/shipwright_build.yaml"}); err != nil {
		return err
	}

	if err := InstallCrossplane(); err != nil {
		return fmt.Errorf("crossplane setup failed: %w", err)
	}

	if err := InstallExternalSecrets(); err != nil {
		return fmt.Errorf("externalsecrets setup failed: %w", err)
	}

	// Needs Shipwright's CRDs (installed above) to already exist.
	fmt.Println("📦 Installing Build Strategies...")
	if err := DependenciesInstall([]string{
		"dependencies/cluster_strategies/buildpacks_v3.yaml",
		"dependencies/cluster_strategies/kaniko.yaml",
		"dependencies/cluster_strategies/buildah_shipwright_managed_push_cr.yaml",
	}); err != nil {
		return err
	}

	// Fetches Flux's CRDs before the Flux manifest below needs them.
	if err := InstallFluxCRDs(); err != nil {
		return err
	}

	fmt.Println("📦 Installing Flux UI + Config...")
	if err := DependenciesInstall([]string{"dependencies/flux/fluxcdui.yaml"}); err != nil {
		return err
	}

	// Needs Crossplane core (installed above via InstallCrossplane) to
	// already be running — its CRD is what this Provider resource uses.
	fmt.Println("📦 Installing Crossplane Github Upjet Provider...")
	if err := DependenciesInstall([]string{"dependencies/crossplane/provider.yaml"}); err != nil {
		return err
	}

	fmt.Println("🎉 BlanketOps environment installed successfully!")
	return nil
}

// UninstallAll tears down the platform stack in the mirror order of
// InstallAll: Helm/script-installed pieces first (RunUninstallScripts),
// then the plain embedded manifests (UninstallManifests), then whatever
// cluster-wide RBAC/CRDs/namespaces are left over (UninstallClusterResources).
func UninstallAll(paths []string) error {
	if err := RunUninstallScripts(); err != nil {
		return fmt.Errorf("uninstall scripts: %w", err)
	}
	if err := UninstallManifests(paths); err != nil {
		return err
	}
	return UninstallClusterResources()
}

// ── Cluster facade ───────────────────────────────────────────────────────
// Thin verbs over the cluster primitives, matching the CLI's command
// surface. Naming: the CLI speaks up/down/status; the package speaks
// Ensure/Delete/Exists.

// ClusterUp ensures the named cluster exists and is ready.
func ClusterUp(name string) error {
	return EnsureCluster(name)
}

// ClusterDown deletes the named cluster.
func ClusterDown(name string) error {
	return DeleteCluster(name)
}

// ClusterStatus reports whether the named cluster exists.
func ClusterStatus(name string) error {
	exists, err := ClusterExists(name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("✅ cluster %q is up\n", name)
	} else {
		fmt.Printf("⭕ cluster %q not found\n", name)
	}
	return nil
}
