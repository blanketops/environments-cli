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

package scripts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// names splits `kubectl get ... -o name` output into individual lines,
// replacing the original script's `for x in $(...)` word-splitting.
func names(resource string, extra ...string) []string {
	args := append([]string{"get", resource, "-o", "name"}, extra...)
	out, err := capture("kubectl", args...)
	if err != nil || out == "" {
		return nil
	}
	var result []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			result = append(result, l)
		}
	}
	return result
}

// Namespaces we touch.
var silenceNamespaces = []string{
	"default",
	"kapp-controller",
	"kapp-controller-packaging-global",
	"argo-events",
	"tekton-pipelines",
	"tekton-pipelines-resolvers",
	"shipwright-build",
	"crossplane-system",
	"external-secrets",
}

// CRD groups we own or install transitively.
var silenceCRDGroupPatterns = []string{
	"blanketops.dev",
	"shipwright.io",
	"tekton.dev",
	"argoproj.io",
	"external-secrets.io",
	"packaging.carvel.dev",
	"kappctrl.k14s.io",
	"sources.toolkit.fluxcd.io",
	"kustomize.toolkit.fluxcd.io",
}

// SilenceFinalizers force-removes finalizers from all BlanketOps-related CRs.
// It is a break-glass tool for a stuck teardown — kept exactly as blunt as the
// original script, so per-object failures are ignored rather than aborting the
// sweep.
func SilenceFinalizers() error {
	if err := requireBinary("kubectl"); err != nil {
		return err
	}

	fmt.Println("🔕 Silencing finalizers across all namespaces...")

	allCRDs := names("crd")

	for _, pattern := range silenceCRDGroupPatterns {
		fmt.Printf("🔎 Processing CRDs matching *.%s\n", pattern)
		for _, crd := range allCRDs {
			if strings.Contains(crd, pattern) {
				fmt.Println(crd)
			}
		}
	}

	for _, ns := range silenceNamespaces {
		fmt.Printf("📂 Namespace: %s\n", ns)
		for _, crd := range allCRDs {
			plural, _ := capture("kubectl", "get", crd, "-o", "jsonpath={.spec.names.plural}")
			group, _ := capture("kubectl", "get", crd, "-o", "jsonpath={.spec.group}")

			if group == "core" || group == "" {
				continue
			}

			for _, obj := range names(plural+"."+group, "-n", ns, "--ignore-not-found") {
				fmt.Printf("  🧨 Removing finalizers from %s\n", obj)
				patch, _ := json.Marshal(map[string]any{
					"metadata": map[string]any{"finalizers": []string{}},
				})
				_ = run("kubectl", "patch", obj, "-n", ns, "--type=merge", "-p", string(patch))
			}
		}
	}

	fmt.Println("✅ Finalizers silenced")
	return nil
}
