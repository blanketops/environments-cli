//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
)

var (
	AppName     = "blanketops-environments"
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

func ensureBin() {
	os.MkdirAll("bin", 0755)
}

func Build() error {
	fmt.Println("🔧 Building", AppName)
	ensureBin()
	err := run("go", "build", "-o", BuildOutput, BuildDir)
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
		"-ldflags", "-s -w",
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
		"-ldflags", "-s -w",
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
