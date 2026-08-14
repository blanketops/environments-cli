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

import "fmt"

// InstallCrossplane installs Crossplane via Helm.
//
// Honours the same env vars as the original script: NAMESPACE, RELEASE,
// VERSION, TIMEOUT. VERSION=latest (any case) resolves to the chart's newest
// release rather than a literal "latest".
func InstallCrossplane() error {
	namespace := getenv("NAMESPACE", "crossplane-system")
	release := getenv("RELEASE", "crossplane")
	repoName := "crossplane-stable"
	repoURL := "https://charts.crossplane.io/stable"
	chart := "crossplane-stable/crossplane"
	version := normalizeHelmVersion(getenv("VERSION", ""))
	timeout := getenv("TIMEOUT", "5m")

	if err := requireBinary("helm"); err != nil {
		return err
	}
	if err := requireBinary("kubectl"); err != nil {
		return err
	}

	fmt.Println("Preparing Helm repo...")

	// reset repo to avoid stale index problems (`|| true` in the original)
	_ = run("helm", "repo", "remove", repoName)

	if err := run("helm", "repo", "add", repoName, repoURL); err != nil {
		return fmt.Errorf("helm repo add failed: %w", err)
	}

	fmt.Println("Updating Helm repositories...")
	if err := runQuiet("helm", "repo", "update"); err != nil {
		return fmt.Errorf("helm repo update failed: %w", err)
	}

	fmt.Println("Installing/upgrading Crossplane...")
	args := []string{
		"upgrade", "--install", release, chart,
		"--namespace", namespace,
		"--create-namespace",
		"--wait",
		"--timeout", timeout,
	}
	if version != "" {
		args = append(args, "--version", version)
	}
	if err := run("helm", args...); err != nil {
		return fmt.Errorf("helm upgrade/install failed: %w", err)
	}

	fmt.Println("Waiting for Crossplane deployment...")
	if err := run("kubectl", "-n", namespace, "rollout", "status", "deployment/crossplane", "--timeout="+timeout); err != nil {
		return fmt.Errorf("rollout status failed: %w", err)
	}

	fmt.Println("Crossplane installation completed.")
	return nil
}
