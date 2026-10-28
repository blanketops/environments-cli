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

// InstallExternalSecrets installs External Secrets via Helm.
//
// Honours the same env vars as the original script: NAMESPACE, RELEASE,
// REPO_NAME, REPO_URL, CHART, TIMEOUT. The chart version is resolved from
// `helm search repo -o json` decoded with encoding/json, so there is no jq
// dependency.
func InstallExternalSecrets() error {
	namespace := getenv("NAMESPACE", "external-secrets")
	release := getenv("RELEASE", "external-secrets")
	repoName := getenv("REPO_NAME", "external-secrets")
	repoURL := getenv("REPO_URL", "https://charts.external-secrets.io")
	chart := getenv("CHART", "external-secrets")
	timeout := getenv("TIMEOUT", "5m")

	if err := requireBinary("helm"); err != nil {
		return err
	}
	if err := requireBinary("kubectl"); err != nil {
		return err
	}

	// --- Helm repo setup ---
	repoListOut, err := capture("helm", "repo", "list", "-o", "json")
	haveRepo := false
	if err == nil {
		var repos []struct {
			Name string `json:"name"`
		}
		if jsonErr := json.Unmarshal([]byte(repoListOut), &repos); jsonErr == nil {
			for _, r := range repos {
				if r.Name == repoName {
					haveRepo = true
					break
				}
			}
		}
	}
	if haveRepo {
		fmt.Printf("✅ Helm repo %s already present\n", repoName)
	} else {
		fmt.Printf("➕ Adding Helm repo %s -> %s\n", repoName, repoURL)
		if err := run("helm", "repo", "add", repoName, repoURL); err != nil {
			return fmt.Errorf("helm repo add failed: %w", err)
		}
	}

	fmt.Println("🔄 Updating Helm repos...")
	if _, err := capture("helm", "repo", "update"); err != nil {
		return fmt.Errorf("helm repo update failed: %w", err)
	}

	// --- Resolve latest chart version ---
	fmt.Println("🔎 Resolving latest External Secrets chart version...")
	searchOut, err := capture("helm", "search", "repo", repoName+"/"+chart, "-o", "json")
	if err != nil {
		return fmt.Errorf("helm search repo failed: %w", err)
	}
	var results []struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(searchOut), &results); err != nil || len(results) == 0 || results[0].Version == "" {
		return fmt.Errorf("unable to resolve External Secrets chart version")
	}
	version := results[0].Version
	fmt.Printf("📦 Latest chart version: %s\n", version)

	// --- Install / Upgrade ---
	fmt.Printf("🚀 Installing External Secrets (release: %s)\n", release)
	if err := run("helm", "upgrade", "--install", release, repoName+"/"+chart,
		"--namespace", namespace,
		"--create-namespace",
		"--wait",
		"--timeout", timeout,
		"--version", version,
	); err != nil {
		return fmt.Errorf("helm upgrade/install failed: %w", err)
	}

	// --- Wait for CRDs ---
	fmt.Println("⏳ Waiting for External Secrets CRDs...")
	crds := []string{
		"externalsecrets.external-secrets.io",
		"clusterexternalsecrets.external-secrets.io",
		"clustersecretstores.external-secrets.io",
		"secretstores.external-secrets.io",
	}
	for _, crd := range crds {
		if err := run("kubectl", "wait", "--for=condition=Established", "crd/"+crd, "--timeout="+timeout); err != nil {
			return fmt.Errorf("kubectl wait failed for %s: %w", crd, err)
		}
	}

	// --- Wait for controller ---
	fmt.Println("⏳ Waiting for External Secrets controller...")
	if err := run("kubectl", "-n", namespace, "rollout", "status", "deployment/external-secrets", "--timeout="+timeout); err != nil {
		return fmt.Errorf("rollout status failed: %w", err)
	}

	// --- Post-install verification ---
	fmt.Println("🔍 Verifying installation...")
	if err := run("kubectl", "get", "pods", "-n", namespace); err != nil {
		return fmt.Errorf("kubectl get pods failed: %w", err)
	}

	fmt.Println("🔍 Checking CRDs...")
	crdListOut, _ := capture("kubectl", "get", "crd")
	for _, line := range strings.Split(crdListOut, "\n") {
		if strings.Contains(line, "external-secrets") {
			fmt.Println(line)
		}
	}

	fmt.Println("🎉 External Secrets installed successfully")
	fmt.Printf("Namespace: %s\n", namespace)
	fmt.Printf("Release:   %s\n", release)
	fmt.Printf("Version:   %s\n", version)
	return nil
}
