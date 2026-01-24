package cmd

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-mvp/cli/core"
)

// ClusterUp ensures a Kind cluster exists and is ready.
func ClusterUp(name string) error {
	if name == "" {
		name = "blanketops"
	}

	return core.EnsureCluster(name)
}

// ClusterDown deletes the Kind cluster.
func ClusterDown(name string) error {
	if name == "" {
		name = "blanketops"
	}

	fmt.Printf("🧨 Deleting cluster %q\n", name)
	return core.DeleteCluster(name)
}

// ClusterStatus reports whether the cluster exists.
func ClusterStatus(name string) error {
	if name == "" {
		name = "blanketops"
	}

	exists, err := core.ClusterExists(name)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("✅ Cluster %q exists\n", name)
	} else {
		fmt.Printf("❌ Cluster %q does not exist\n", name)
	}

	return nil
}
