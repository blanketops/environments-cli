package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runDist() error {
	// Resolve project root
	root, err := filepath.Abs("../operator")
	if err != nil {
		return fmt.Errorf("failed to resolve operator directory: %w", err)
	}

	// Paths
	configDir := filepath.Join(root, "config", "default")
	outputDir := filepath.Join(root, "dist")
	outputFile := filepath.Join(outputDir, "install.yaml")

	// Ensure dist directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create dist: %w", err)
	}

	// Kustomize build
	cmd := exec.Command("kustomize", "build", configDir)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	// Write output to dist/install.yaml
	if err := os.WriteFile(outputFile, out, 0o644); err != nil {
		return fmt.Errorf("failed to write install.yaml: %w", err)
	}

	fmt.Println("✅ dist/install.yaml generated successfully")
	return nil
}
