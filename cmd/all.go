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
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// manifestPaths walks the embedded dependencies tree and returns every
// manifest, sorted for a stable apply order. The embed is the single
// source of truth — no hand-maintained path lists to drift.
func manifestPaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(Assets, "dependencies", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

// InstallAll applies every embedded dependency manifest.
func InstallAll() error {
	paths, err := manifestPaths()
	if err != nil {
		return err
	}
	return DependenciesInstall(paths)
}

// UninstallAll removes the given manifests, or — when called with nil —
// every embedded dependency manifest in reverse apply order.
func UninstallAll(paths []string) error {
	if paths == nil {
		all, err := manifestPaths()
		if err != nil {
			return err
		}
		// reverse: tear down in the opposite order of install
		for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
			all[i], all[j] = all[j], all[i]
		}
		paths = all
	}
	return UninstallManifests(paths)
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

var _ = strings.TrimSpace // placeholder if unused; remove
