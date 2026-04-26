package core

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// DependenciesInstall (uses embedded assets)
// ---------------------------------------------------------------------------
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

		data, err := Assets.ReadFile(path)
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
// ApplyEmbeddedDir
// ---------------------------------------------------------------------------
func ApplyEmbeddedDir(path string) error {

	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	entries, err := Assets.ReadDir(path)
	if err != nil {
		return fmt.Errorf("embedded path not found: %w", err)
	}

	for _, e := range entries {

		if e.IsDir() {
			continue
		}

		file := filepath.Join(path, e.Name())

		data, err := Assets.ReadFile(file)
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

// ---------------------------------------------------------------------------
// Flux CRDs
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
// Script runner (embedded)
// ---------------------------------------------------------------------------
func runScript(path string, args ...string) error {

	data, err := Assets.ReadFile(path)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "blanketops-script-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return err
	}

	tmp.Close()

	cmd := exec.Command("bash", append([]string{tmp.Name()}, args...)...)
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
