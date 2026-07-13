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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const latestReleaseAPI = "https://api.github.com/repos/blanketops/environments-cli/releases/latest"

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

// releaseAssetName returns the static-binary asset name for the running
// architecture. Only linux is published today — that's all release.yml
// builds — so anything else is a clear error rather than a confusing
// download failure.
func releaseAssetName() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("self-install only supports linux today (running %s)", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64":
		return "bops-env-static", nil
	case "arm64":
		return "bops-env-static-arm64", nil
	default:
		return "", fmt.Errorf("self-install has no published binary for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func fetchLatestRelease() (*ghRelease, error) {
	resp, err := http.Get(latestReleaseAPI)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch latest release: %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

func findAsset(rel *ghRelease, name string) (*ghAsset, bool) {
	for i := range rel.Assets {
		if rel.Assets[i].Name == name {
			return &rel.Assets[i], true
		}
	}
	return nil, false
}

func downloadTo(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// SelfInstall fetches, verifies, and installs the latest bops-env release.
func SelfInstall() error {
	assetName, err := releaseAssetName()
	if err != nil {
		return err
	}

	fmt.Println("🔎 Resolving latest release...")
	rel, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	fmt.Printf("📦 Latest release: %s\n", rel.TagName)

	asset, ok := findAsset(rel, assetName)
	if !ok {
		return fmt.Errorf("release %s has no asset named %s", rel.TagName, assetName)
	}

	tmpDir, err := os.MkdirTemp("", "bops-env-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, assetName)
	fmt.Printf("⬇️  Downloading %s...\n", asset.Name)
	if err := downloadTo(asset.BrowserDownloadURL, binPath); err != nil {
		return err
	}

	if sigAsset, ok := findAsset(rel, assetName+".sig"); ok {
		sigPath := binPath + ".sig"
		if err := downloadTo(sigAsset.BrowserDownloadURL, sigPath); err != nil {
			return fmt.Errorf("download signature: %w", err)
		}
		if err := verifySignature(binPath, sigPath); err != nil {
			return fmt.Errorf("signature verification failed — refusing to install: %w", err)
		}
	} else {
		fmt.Println("⚠️  No signature asset found for this release — installing unverified")
	}

	return installBinary(binPath)
}

// verifySignature runs cosign verify-blob if cosign is available. Missing
// cosign is a warning, not a hard failure — verification is best-effort
// for a self-install convenience command, not the release pipeline. An
// actual verification failure (signature present but invalid) always
// blocks the install.
func verifySignature(binPath, sigPath string) error {
	if _, err := exec.LookPath("cosign"); err != nil {
		fmt.Println("⚠️  cosign not found on PATH — skipping signature verification")
		return nil
	}
	cmd := exec.Command("cosign", "verify-blob",
		"--certificate-identity-regexp", ".*",
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		"--signature", sigPath,
		binPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("✅ Signature verified")
	return nil
}

// installBinary copies the downloaded binary into ~/.local/bin, falling
// back to ~/bin if the former is mounted noexec — mirrors magefile.go's
// Install() target, duplicated here because magefile.go is excluded from
// normal builds (//go:build mage) and isn't linked into this binary.
func installBinary(binPath string) error {
	installDir := os.Getenv("HOME") + "/.local/bin"
	fallbackDir := os.Getenv("HOME") + "/bin"
	target := installDir + "/bops-env"

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o755); err != nil {
		return err
	}

	fmt.Println("🔍 Testing executability")
	testFile := installDir + "/.__exec_test"
	os.WriteFile(testFile, []byte("#!/bin/sh\necho test_ok\n"), 0o755)
	testErr := exec.Command(testFile).Run()
	os.Remove(testFile)

	if testErr != nil {
		fmt.Println("⚠️ noexec detected — switching to", fallbackDir)
		if err := os.MkdirAll(fallbackDir, 0o755); err != nil {
			return err
		}
		target = fallbackDir + "/bops-env"
		if err := os.WriteFile(target, data, 0o755); err != nil {
			return err
		}
		fmt.Println("🎉 Installed to", target)
		fmt.Println("ℹ️ Add to PATH:")
		fmt.Println("export PATH=\"" + fallbackDir + ":$PATH\"")
		return nil
	}

	fmt.Println("✅ Installed to", target)
	return nil
}

// SelfUninstall removes a self-installed bops-env binary from either of
// the locations installBinary might have used.
func SelfUninstall() error {
	installDir := os.Getenv("HOME") + "/.local/bin"
	fallbackDir := os.Getenv("HOME") + "/bin"

	removed := false
	for _, path := range []string{installDir + "/bops-env", fallbackDir + "/bops-env"} {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}
			fmt.Println("🗑️  Removed", path)
			removed = true
		}
	}
	if !removed {
		fmt.Println("ℹ️ No self-installed bops-env binary found")
	}
	return nil
}
