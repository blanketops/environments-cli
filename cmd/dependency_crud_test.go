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
	"io"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. The functions under test here (ListDependencies,
// printDependencyStatus) print directly rather than returning a string, so
// this is the only way to assert on their output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(out)
}

func TestListDependencies(t *testing.T) {
	out := captureStdout(t, ListDependencies)
	for _, d := range registry {
		if !strings.Contains(out, d.Name) {
			t.Errorf("ListDependencies() output missing dependency name %q\noutput:\n%s", d.Name, out)
		}
	}
}

func TestPrintDependencyStatus(t *testing.T) {
	cases := []struct {
		name      string
		dep       *Dependency
		installed bool
		want      string
	}{
		{
			name:      "no namespace to check",
			dep:       &Dependency{Name: "buildstrategies"},
			installed: false,
			want:      "❔",
		},
		{
			name:      "namespace present",
			dep:       &Dependency{Name: "carvel", Namespace: "kapp-controller"},
			installed: true,
			want:      "✅",
		},
		{
			name:      "namespace absent",
			dep:       &Dependency{Name: "knative", Namespace: "knative-serving"},
			installed: false,
			want:      "⭕",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStdout(t, func() { printDependencyStatus(tc.dep, tc.installed) })
			if !strings.Contains(out, tc.want) {
				t.Errorf("printDependencyStatus(%+v, %v) output = %q, want it to contain %q", tc.dep, tc.installed, out, tc.want)
			}
			if !strings.Contains(out, tc.dep.Name) {
				t.Errorf("printDependencyStatus(%+v, %v) output = %q, want it to contain the dependency name", tc.dep, tc.installed, out)
			}
		})
	}
}

// fakeNamespace builds an unstructured Namespace object, the shape the
// fake dynamic client and dependencyInstalled's namespaceGVR lookup expect.
func fakeNamespace(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

func TestDependencyInstalled(t *testing.T) {
	dc := fake.NewSimpleDynamicClient(runtime.NewScheme(), fakeNamespace("kapp-controller"))

	t.Run("no namespace configured", func(t *testing.T) {
		installed, err := dependencyInstalled(dc, &Dependency{Name: "buildstrategies"})
		if err != nil {
			t.Fatalf("dependencyInstalled returned error: %v", err)
		}
		if installed {
			t.Error("dependencyInstalled with no Namespace field = true, want false")
		}
	})

	t.Run("namespace exists on cluster", func(t *testing.T) {
		installed, err := dependencyInstalled(dc, &Dependency{Name: "carvel", Namespace: "kapp-controller"})
		if err != nil {
			t.Fatalf("dependencyInstalled returned error: %v", err)
		}
		if !installed {
			t.Error("dependencyInstalled for a namespace present on the cluster = false, want true")
		}
	})

	t.Run("namespace missing from cluster", func(t *testing.T) {
		installed, err := dependencyInstalled(dc, &Dependency{Name: "knative", Namespace: "knative-serving"})
		if err != nil {
			t.Fatalf("dependencyInstalled returned error: %v", err)
		}
		if installed {
			t.Error("dependencyInstalled for a namespace absent from the cluster = true, want false")
		}
	})
}

func TestInstallDependency_UnknownName(t *testing.T) {
	if err := InstallDependency("nonexistent-dependency"); err == nil {
		t.Fatal("InstallDependency(unknown name) returned nil error, want a non-nil error — it must fail before touching the cluster")
	}
}

func TestUninstallDependency_UnknownName(t *testing.T) {
	if err := UninstallDependency("nonexistent-dependency"); err == nil {
		t.Fatal("UninstallDependency(unknown name) returned nil error, want a non-nil error — it must fail before touching the cluster")
	}
}
