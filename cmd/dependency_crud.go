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
	"os"
	"os/exec"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var namespaceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

// ListDependencies prints every dependency this CLI can address
// individually, one per line.
func ListDependencies() {
	for _, d := range registry {
		fmt.Printf("  %-18s %s\n", d.Name, d.Description)
	}
}

// InstallDependency installs (or, since the underlying apply is
// create-or-update, re-applies/updates) a single named dependency.
func InstallDependency(name string) error {
	dep, err := FindDependency(name)
	if err != nil {
		return err
	}
	fmt.Printf("📦 Installing %s...\n", dep.Name)
	if dep.install != nil {
		return dep.install()
	}
	return DependenciesInstall(dep.Manifests)
}

// UninstallDependency removes a single named dependency's resources.
func UninstallDependency(name string) error {
	dep, err := FindDependency(name)
	if err != nil {
		return err
	}
	fmt.Printf("🗑️  Uninstalling %s...\n", dep.Name)
	if dep.uninstall != nil {
		return dep.uninstall()
	}
	if len(dep.Manifests) == 0 {
		return fmt.Errorf("dependency %q has no manifests to uninstall", dep.Name)
	}
	return UninstallManifests(dep.Manifests)
}

// StatusDependency reports whether a single named dependency looks
// installed, based on its namespace existing on the cluster. This is a
// coarse signal (a namespace can exist with a partially-failed install),
// not a full health check.
func StatusDependency(name string) error {
	dep, err := FindDependency(name)
	if err != nil {
		return err
	}
	dc, _, err := NewDynamicClient()
	if err != nil {
		return err
	}
	installed, err := dependencyInstalled(dc, dep)
	if err != nil {
		return err
	}
	printDependencyStatus(dep, installed)
	return nil
}

// StatusAll reports the status of every registered dependency, reusing a
// single client for all of them rather than dialing the cluster per entry.
func StatusAll() error {
	dc, _, err := NewDynamicClient()
	if err != nil {
		return err
	}
	for i := range registry {
		dep := &registry[i]
		installed, err := dependencyInstalled(dc, dep)
		if err != nil {
			return err
		}
		printDependencyStatus(dep, installed)
	}
	return nil
}

func printDependencyStatus(dep *Dependency, installed bool) {
	if dep.Namespace == "" {
		fmt.Printf("❔ %-18s status unknown (no namespace to check)\n", dep.Name)
		return
	}
	if installed {
		fmt.Printf("✅ %-18s installed (namespace %q present)\n", dep.Name, dep.Namespace)
	} else {
		fmt.Printf("⭕ %-18s not installed (namespace %q not found)\n", dep.Name, dep.Namespace)
	}
}

// dependencyInstalled reports whether dep's namespace exists, via the
// given client — injectable so tests can pass a fake dynamic.Interface
// instead of dialing a real cluster.
func dependencyInstalled(dc dynamic.Interface, dep *Dependency) (bool, error) {
	if dep.Namespace == "" {
		return false, nil
	}
	_, err := dc.Resource(namespaceGVR).Get(context.Background(), dep.Namespace, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

// helmUninstall runs `helm uninstall <release> -n <namespace>`, used by
// registry entries for Helm-installed dependencies.
func helmUninstall(release, namespace string) error {
	cmd := exec.Command("helm", "uninstall", release, "-n", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
