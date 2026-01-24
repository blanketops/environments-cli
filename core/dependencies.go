package core

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// DependenciesInstall
// ---------------------------------------------------------------------------
// DependenciesInstall reads YAML manifests from disk and applies them to the cluster.
func DependenciesInstall(paths []string) error {
	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	for _, path := range paths {
		fmt.Printf("📄 Applying %s\n", path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read manifest %s: %w", path, err)
		}

		objs, err := decodeYAMLStream(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}

		if err := robustApply(dc, mapper, objs); err != nil {
			return fmt.Errorf("apply %s: %w", path, err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// ApplyFromURL
// ---------------------------------------------------------------------------
func ApplyFromURL(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch url %s: %w", url, err)
	}
	defer resp.Body.Close()

	objs, err := decodeYAMLDocuments(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode YAML from %s: %w", url, err)
	}

	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	for _, o := range objs {
		if err := applyUnstructured(dc, mapper, o); err != nil {
			return err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// ApplyLocalDir (filesystem only)
// ---------------------------------------------------------------------------
func ApplyLocalDir(path string) error {
	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %w", err)
	}

	if stat.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			file := filepath.Join(path, e.Name())
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			objs, err := decodeYAMLStream(bytes.NewReader(data))
			if err != nil {
				return err
			}

			if err := robustApply(dc, mapper, objs); err != nil {
				return err
			}
		}
		return nil
	}

	// single file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	objs, err := decodeYAMLStream(bytes.NewReader(data))
	if err != nil {
		return err
	}

	return robustApply(dc, mapper, objs)
}

// ---------------------------------------------------------------------------
// Flux CRDs (URL-based)
// ---------------------------------------------------------------------------
func InstallFluxCRDs() error {
	fmt.Println("📘 Installing Flux CRDs...")

	url := "https://github.com/fluxcd/flux2/releases/latest/download/install.yaml"
	if err := ApplyFromURL(url); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)
	return nil
}

// ---------------------------------------------------------------------------
// Script helpers (filesystem)
// ---------------------------------------------------------------------------
func runScript(path string, args ...string) error {
	cmd := exec.Command("bash", append([]string{path}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func InstallCrossplane() error {
	fmt.Println("🔐 Running Crossplane Setup Script...")
	return runScript("scripts/install-crossplane.sh")
}

func InstallExternalSecrets() error {
	fmt.Println("🔐 Running ExternalSecrets Setup Script...")
	return runScript("scripts/install-externalsecrets.sh")
}

func RunShipwrightCertSetup() error {
	fmt.Println("🔐 Running Shipwright Certificate Setup Script...")
	return runScript("scripts/setup-shipwright-cert.sh")
}
