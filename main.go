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
	"strings"

	"github.com/blanketops/environments-cli/cmd"
)

// uninstallPaths is the manifest set removed on uninstall, in reverse
// dependency order. Must stay in sync with InstallAll's explicit sequence
// in cmd/all.go.
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
	printGrid([][2]string{
		{"Binary", "bops-env"},
		{"Repo", "github.com/blanketops/environments-cli"},
	})
}

// printGrid renders a two-column bordered table. Column widths are
// computed from content, not hand-counted, so the borders always line up
// regardless of what's in the cells.
func printGrid(rows [][2]string) {
	col1, col2 := 0, 0
	for _, r := range rows {
		if len(r[0]) > col1 {
			col1 = len(r[0])
		}
		if len(r[1]) > col2 {
			col2 = len(r[1])
		}
	}

	top := "┌" + strings.Repeat("─", col1+2) + "┬" + strings.Repeat("─", col2+2) + "┐"
	mid := "├" + strings.Repeat("─", col1+2) + "┼" + strings.Repeat("─", col2+2) + "┤"
	bot := "└" + strings.Repeat("─", col1+2) + "┴" + strings.Repeat("─", col2+2) + "┘"

	fmt.Println(top)
	for i, r := range rows {
		fmt.Printf("│ %-*s │ %-*s │\n", col1, r[0], col2, r[1])
		if i < len(rows)-1 {
			fmt.Println(mid)
		}
	}
	fmt.Println(bot)
}

// usageEntry pairs a command invocation with a one-line description of
// what it does, so `bops-env` with no args is self-documenting.
type usageEntry struct {
	cmd  string
	desc string
}

var usageEntries = [][]usageEntry{
	{
		{"bops-env install", "Fetch and install the latest bops-env release"},
		{"bops-env uninstall", "Remove a self-installed bops-env release"},
		{"bops-env dist", "Reserved (not yet implemented)"},
	},
	{
		{"bops-env dependencies install", "Install the platform stack (all dependency manifests)"},
		{"bops-env dependencies uninstall", "Remove the platform stack"},
	},
	{
		{"bops-env cluster up [name]", "Create the named cluster if it doesn't exist"},
		{"bops-env cluster down [name]", "Delete the named cluster"},
		{"bops-env cluster status [name]", "Show whether the named cluster is up"},
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
	// INSTALL / UNINSTALL — self-install: fetches and installs the latest
	// bops-env release from GitHub. The platform stack (the embedded
	// dependencies/ tree) installs via `dependencies install`/
	// `dependencies uninstall` below — that's a separate concern from
	// installing the CLI binary itself.
	// ---------------------------------------------------------------------
	case "install":
		if err := cmd.SelfInstall(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := cmd.SelfUninstall(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
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
