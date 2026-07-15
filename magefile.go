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
	"strings"
)

var (
	AppName     = "bops-env"
	BuildDir    = "."
	BuildOutput = "bin/" + AppName
	StaticOut   = "bin/" + AppName + "-static"
	InstallDir  = os.Getenv("HOME") + "/.local/bin"
	FallbackDir = os.Getenv("HOME") + "/bin"
)

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
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
	return run("go", "test", "./...")
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
	fmt.Println("🔎 Verifying static binary")
	run("ldd", StaticOut)
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
	target := InstallDir + "/" + AppName
	err := run("cp", BuildOutput, target)
	if err != nil {
		return err
	}
	run("chmod", "+x", target)
	fmt.Println("🔍 Testing executability")
	testFile := InstallDir + "/.__exec_test"
	os.WriteFile(testFile, []byte("#!/bin/sh\necho test_ok\n"), 0755)
	err = run(testFile)
	os.Remove(testFile)
	if err != nil {
		fmt.Println("⚠️ noexec detected — switching to", FallbackDir)
		os.MkdirAll(FallbackDir, 0755)
		target = FallbackDir + "/" + AppName
		run("cp", BuildOutput, target)
		run("chmod", "+x", target)
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
	os.Remove(InstallDir + "/" + AppName)
	os.Remove(FallbackDir + "/" + AppName)
	run("sudo", "rm", "-f", "/usr/local/bin/"+AppName)
	os.Remove("bin/" + AppName)
	os.Remove("bin/" + AppName + "-static")
	os.Remove("bin/" + AppName + "-static-arm64")
	fmt.Println("✔️ All copies removed")
	return nil
}

func Clean() {
	fmt.Println("🧹 Cleaning build artifacts")
	os.RemoveAll("bin")
}
