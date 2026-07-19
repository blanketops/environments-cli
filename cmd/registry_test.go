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

import "testing"

func TestFindDependency(t *testing.T) {
	dep, err := FindDependency("carvel")
	if err != nil {
		t.Fatalf("FindDependency(%q) returned error: %v", "carvel", err)
	}
	if dep.Name != "carvel" {
		t.Fatalf("FindDependency(%q).Name = %q, want %q", "carvel", dep.Name, "carvel")
	}
}

func TestFindDependency_Unknown(t *testing.T) {
	_, err := FindDependency("nonexistent-dependency")
	if err == nil {
		t.Fatal("FindDependency(unknown name) returned nil error, want a non-nil error naming the unknown dependency")
	}
}

func TestDependencyNames(t *testing.T) {
	names := DependencyNames()
	if len(names) != len(registry) {
		t.Fatalf("DependencyNames() returned %d names, want %d (one per registry entry)", len(names), len(registry))
	}
	for i, d := range registry {
		if names[i] != d.Name {
			t.Errorf("DependencyNames()[%d] = %q, want %q", i, names[i], d.Name)
		}
	}
}

// TestRegistryIntegrity guards the invariants InstallDependency and
// UninstallDependency rely on: every entry must have a name, a unique
// name, and a way to actually install and uninstall it — either an
// explicit override or a non-empty manifest list (the default install
// path silently no-ops on an empty list, and the default uninstall path
// errors outright).
func TestRegistryIntegrity(t *testing.T) {
	seen := make(map[string]bool, len(registry))
	for _, d := range registry {
		if d.Name == "" {
			t.Fatalf("registry has an entry with an empty Name (Description %q)", d.Description)
		}
		if seen[d.Name] {
			t.Errorf("registry has duplicate entry for name %q", d.Name)
		}
		seen[d.Name] = true

		if d.install == nil && len(d.Manifests) == 0 {
			t.Errorf("dependency %q has no install override and no Manifests — InstallDependency would silently no-op", d.Name)
		}
		if d.uninstall == nil && len(d.Manifests) == 0 {
			t.Errorf("dependency %q has no uninstall override and no Manifests — UninstallDependency would always error", d.Name)
		}
	}
}
