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
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/blanketops/environments-cli/util"
)

var Assets embed.FS

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
		p := util.NewSpinner(path)
		data, err := Assets.ReadFile(path)
		if err != nil {
			err = fmt.Errorf("read manifest %s: %w", path, err)
			p.Fail(err)
			return err
		}
		objs, err := decodeYAMLStream(bytes.NewReader(data))
		if err != nil {
			err = fmt.Errorf("decode %s: %w", path, err)
			p.Fail(err)
			return err
		}
		if err := robustApply(dc, mapper, objs, p); err != nil {
			err = fmt.Errorf("apply %s: %w", path, err)
			p.Fail(err)
			return err
		}
		p.Done("")
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

	p := util.NewSpinner(url)
	for _, o := range objs {
		gvk := o.GetObjectKind().GroupVersionKind()
		p.Update(fmt.Sprintf("%s %s", gvk.Kind, o.GetName()))
		if err := applyUnstructured(dc, mapper, o); err != nil {
			p.Fail(err)
			return err
		}
	}
	p.Done("")
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
		p := util.NewSpinner(file)
		data, err := Assets.ReadFile(file)
		if err != nil {
			p.Fail(err)
			return err
		}
		objs, err := decodeYAMLStream(bytes.NewReader(data))
		if err != nil {
			p.Fail(err)
			return err
		}
		if err := robustApply(dc, mapper, objs, p); err != nil {
			p.Fail(err)
			return err
		}
		p.Done("")
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
	tmp, err := os.CreateTemp("", "environments-script-*")
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
