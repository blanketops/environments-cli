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

	"github.com/ntlaletsi70/blanketops-environments-cli/cmd"
)

// uninstallPaths is the manifest set removed on uninstall, in reverse
// dependency order. Must stay in sync with the embedded dependencies tree.
var uninstallPaths = []string{
	"dependencies/carvel/release.yaml",
	"dependencies/argoevents/manifest.yaml",
	"dependencies/tekton/tekton_pipelines.yaml",
	"dependencies/tekton/tekton_dashboard.yaml",
	"dependencies/shipwright/shipwright_build.yaml",
	"dependencies/flux/fluxcdui.yaml",
	"dependencies/cluster_strategies/buildpacks_v3.yaml",
	"dependencies/cluster_strategies/kaniko.yaml",
	"dependencies/cluster_strategies/buildah_shipwright_managed_push_cr.yaml",
	"dependencies/crossplane/provider.yaml",
}

func banner() {
	fmt.Println("🚀 BlanketOps Environments CLI")
	fmt.Println("──────────────────────────────────────────")
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("")
	fmt.Println("  bops install")
	fmt.Println("  bops uninstall")
	fmt.Println("  bops release apply")
	fmt.Println("")
	fmt.Println("  bops dependencies install")
	fmt.Println("  bops dependencies uninstall")
	fmt.Println("")
	fmt.Println("  bops cluster up [name]")
	fmt.Println("  bops cluster down [name]")
	fmt.Println("  bops cluster status [name]")
	fmt.Println("")
}

func main() {
	// Wire the embedded assets into the cmd package before anything can
	// read them — cmd.Assets is a zero-value embed.FS until this runs.
	cmd.Assets = Assets

	banner()
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	// ---------------------------------------------------------------------
	// INSTALL / UNINSTALL
	// ---------------------------------------------------------------------
	case "install":
		if err := cmd.InstallAll(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := cmd.UninstallAll(uninstallPaths); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "dist":
		fmt.Println("ℹ️ dist command reserved (no-op for now)")
		return
	// ---------------------------------------------------------------------
	// DEPENDENCIES
	// ---------------------------------------------------------------------
	case "dependencies":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		switch os.Args[2] {
		case "install":
			if err := cmd.InstallAll(); err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "uninstall":
			if err := cmd.UninstallAll(uninstallPaths); err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		default:
			fmt.Println("Unknown dependencies command:", os.Args[2])
			usage()
			os.Exit(1)
		}
	// ---------------------------------------------------------------------
	// CLUSTER
	// ---------------------------------------------------------------------
	case "cluster":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		name := ""
		if len(os.Args) >= 4 {
			name = os.Args[3]
		}
		switch os.Args[2] {
		case "up":
			if err := cmd.ClusterUp(name); err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "down":
			if err := cmd.ClusterDown(name); err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "status":
			if err := cmd.ClusterStatus(name); err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		default:
			fmt.Println("Unknown cluster command:", os.Args[2])
			usage()
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		usage()
		os.Exit(1)
	}
}
