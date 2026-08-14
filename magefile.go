//go:build mage

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

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// exeSuffix returns ".exe" on Windows, where a binary won't run via
// implicit PATH/extension lookup without it, and "" everywhere else.
func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// userHomeDir wraps os.UserHomeDir (which checks %USERPROFILE% on
// Windows, $HOME elsewhere) with the same $HOME fallback os.UserHomeDir
// itself would use if that lookup ever fails.
func userHomeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

var (
	AppName     = "bops-env"
	BuildDir    = "."
	BuildOutput = "bin/" + AppName + exeSuffix()
	StaticOut   = "bin/" + AppName + "-static"
	InstallDir  = filepath.Join(userHomeDir(), ".local", "bin")
	FallbackDir = filepath.Join(userHomeDir(), "bin")
)

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// copyFile copies src to dst byte-for-byte (overwriting dst) and marks it
// executable, in place of shelling out to `cp`/`chmod` — neither exists
// natively on Windows, unlike Linux/macOS where both are always present.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}

// version resolves the release tag to stamp into the binary: the VERSION
// env var if set (release.yml exports github.ref_name), otherwise
// `git describe` for local dev builds, otherwise "dev".
func version() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// ldflags stamps main.version via -X so `bops-env`'s banner can print the
// version it was actually built from.
func ldflags() string {
	return "-X main.version=" + version()
}

func ensureBin() {
	os.MkdirAll("bin", 0755)
}

func Vet() error {
	fmt.Println("🔍 Vetting", AppName)
	return run("go", "vet", "./...")
}

func Test() error {
	fmt.Println("🧪 Testing", AppName)
	return run("go", "test", "-cover", "./...")
}

// Coverage writes a full coverage profile to bin/coverage.out and renders
// it as HTML at bin/coverage.html, so gaps can be inspected file-by-file
// instead of just the per-package percentage Test() prints.
func Coverage() error {
	fmt.Println("🧪 Testing", AppName, "with coverage report")
	ensureBin()
	profile := "bin/coverage.out"
	if err := run("go", "test", "-coverprofile="+profile, "./..."); err != nil {
		return err
	}
	if err := run("go", "tool", "cover", "-func="+profile); err != nil {
		return err
	}
	html := "bin/coverage.html"
	if err := run("go", "tool", "cover", "-html="+profile, "-o", html); err != nil {
		return err
	}
	fmt.Println("✅ Coverage report:", html)
	return nil
}

func Build() error {
	fmt.Println("🔧 Building", AppName)
	ensureBin()
	err := run("go", "build", "-ldflags", ldflags(), "-o", BuildOutput, BuildDir)
	if err != nil {
		return err
	}
	fmt.Println("✅ Build complete:", BuildOutput)
	return nil
}

func Static() error {
	fmt.Println("🏗 Building static binary")
	ensureBin()
	cmd := exec.Command("go", "build",
		"-trimpath",
		"-ldflags", "-s -w "+ldflags(),
		"-o", StaticOut,
		BuildDir,
	)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		fmt.Println("🔎 Verifying static binary")
		run("ldd", StaticOut)
	} else {
		// ldd isn't available on macOS/Windows hosts — this cross-compiled
		// linux binary can't be verified locally on those, only built.
		fmt.Println("ℹ️ skipping ldd check (not available on", runtime.GOOS+")")
	}
	fmt.Println("➡ Ready for gokrazy")
	return nil
}

func StaticArm64() error {
	fmt.Println("🏗 Building static ARM64")
	ensureBin()
	cmd := exec.Command("go", "build",
		"-ldflags", "-s -w "+ldflags(),
		"-o", "bin/"+AppName+"-static-arm64",
		BuildDir,
	)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=arm64",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("➡ Ready for gokrazy")
	return nil
}

func Install() error {
	fmt.Println("📦 Installing", AppName)
	if err := Build(); err != nil {
		return err
	}
	os.MkdirAll(InstallDir, 0755)
	target := filepath.Join(InstallDir, filepath.Base(BuildOutput))
	if err := copyFile(BuildOutput, target); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// Windows has no noexec-mount concept to probe for below, so
		// there's no fallback-directory case to check — installing to
		// InstallDir is the whole story.
		fmt.Println("✅ Installed to", target)
		fmt.Println("ℹ️ Add this to your PATH if it isn't already:")
		fmt.Println("  ", InstallDir)
		return nil
	}

	fmt.Println("🔍 Testing executability")
	testFile := filepath.Join(InstallDir, ".__exec_test")
	os.WriteFile(testFile, []byte("#!/bin/sh\necho test_ok\n"), 0755)
	err := run(testFile)
	os.Remove(testFile)
	if err != nil {
		fmt.Println("⚠️ noexec detected — switching to", FallbackDir)
		os.MkdirAll(FallbackDir, 0755)
		target = filepath.Join(FallbackDir, filepath.Base(BuildOutput))
		copyFile(BuildOutput, target)
		fmt.Println("🎉 Installed to", target)
		fmt.Println("ℹ️ Add to PATH:")
		fmt.Println("export PATH=\"" + FallbackDir + ":$PATH\"")
	} else {
		fmt.Println("✅ Installed to", target)
	}
	return nil
}

func Uninstall() error {
	fmt.Println("🧹 Uninstalling", AppName)
	name := filepath.Base(BuildOutput)
	os.Remove(filepath.Join(InstallDir, name))
	os.Remove(filepath.Join(FallbackDir, name))
	if runtime.GOOS != "windows" {
		run("sudo", "rm", "-f", "/usr/local/bin/"+AppName)
	}
	os.Remove(BuildOutput)
	os.Remove(StaticOut)
	os.Remove("bin/" + AppName + "-static-arm64")
	fmt.Println("✔️ All copies removed")
	return nil
}

func Clean() {
	fmt.Println("🧹 Cleaning build artifacts")
	os.RemoveAll("bin")
}
