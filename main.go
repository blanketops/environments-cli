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

// usageEntry pairs a command invocation with a one-line description of
// what it does, so `bops` with no args is self-documenting.
type usageEntry struct {
	cmd  string
	desc string
}

var usageEntries = [][]usageEntry{
	{
		{"bops install", "Fetch and install the latest bops release (reserved, not yet implemented)"},
		{"bops uninstall", "Remove a self-installed bops release (reserved, not yet implemented)"},
		{"bops dist", "Reserved (not yet implemented)"},
	},
	{
		{"bops dependencies install", "Install the platform stack (all dependency manifests)"},
		{"bops dependencies uninstall", "Remove the platform stack"},
	},
	{
		{"bops cluster up [name]", "Create the named cluster if it doesn't exist"},
		{"bops cluster down [name]", "Delete the named cluster"},
		{"bops cluster status [name]", "Show whether the named cluster is up"},
	},
}

func usage() {
	width := 0
	for _, group := range usageEntries {
		for _, e := range group {
			if len(e.cmd) > width {
				width = len(e.cmd)
			}
		}
	}

	fmt.Println("Usage:")
	fmt.Println("")
	for _, group := range usageEntries {
		for _, e := range group {
			fmt.Printf("  %-*s  %s\n", width, e.cmd, e.desc)
		}
		fmt.Println("")
	}
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
	// INSTALL / UNINSTALL — reserved for self-install: fetching and
	// installing the latest bops release/dist, the same job "dist" was
	// already reserved for. Not implemented yet. The platform stack
	// (the embedded dependencies/ tree) installs via `dependencies
	// install`/`dependencies uninstall` below.
	// ---------------------------------------------------------------------
	case "install":
		fmt.Println("ℹ️ install is reserved for fetching the latest bops release (not yet implemented)")
		fmt.Println("ℹ️ Use `bops dependencies install` to install the platform stack")
		return
	case "uninstall":
		fmt.Println("ℹ️ uninstall is reserved for removing a self-installed bops release (not yet implemented)")
		fmt.Println("ℹ️ Use `bops dependencies uninstall` to remove the platform stack")
		return
	case "dist":
		fmt.Println("ℹ️ dist command reserved (no-op for now)")
		return
	// ---------------------------------------------------------------------
	// DEPENDENCIES — installs/removes the platform stack: every manifest
	// under the embedded dependencies/ tree.
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
