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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// EnsureCluster ensures a Kubernetes cluster exists and is ready.
// For MVP, this is Kind-only.
func EnsureCluster(name string) error {
	exists, err := ClusterExists(name)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("✅ Cluster %q already exists\n", name)
		return ensureClusterReady()
	}

	fmt.Printf("🚀 Creating cluster %q\n", name)
	if err := CreateCluster(name); err != nil {
		return err
	}

	return ensureClusterReady()
}

// CreateCluster creates a Kind cluster.
func CreateCluster(name string) error {
	if _, err := exec.LookPath("kind"); err != nil {
		return fmt.Errorf("kind not found in PATH")
	}

	cmd := exec.Command("kind", "create", "cluster", "--name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DeleteCluster deletes a Kind cluster.
func DeleteCluster(name string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ClusterExists checks if a Kind cluster exists.
func ClusterExists(name string) (bool, error) {
	cmd := exec.Command("kind", "get", "clusters")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to list kind clusters: %w", err)
	}

	for _, line := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}

	return false, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func ensureClusterReady() error {
	fmt.Println("⏳ Waiting for Kubernetes API...")

	// allow kubeconfig to settle
	time.Sleep(2 * time.Second)

	if err := ensureKubectl(); err != nil {
		return err
	}

	fmt.Println("🔎 Waiting for nodes to become Ready...")
	if err := WaitForAllNodesReady(); err != nil {
		return err
	}

	fmt.Println("✅ Cluster is ready")
	return nil
}

func ensureKubectl() error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found in PATH")
	}

	cmd := exec.Command("kubectl", "version", "--short")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
