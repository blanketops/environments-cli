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
	"dependencies/kourier/kourier.yaml",
	"dependencies/knative/serving-core.yaml",
	"dependencies/knative/serving-crds.yaml",
	"dependencies/cluster_strategies/buildpacks_v3.yaml",
	"dependencies/cluster_strategies/kaniko.yaml",
	"dependencies/cluster_strategies/buildah_shipwright_managed_push_cr.yaml",
	"dependencies/crossplane/provider.yaml",
}

// version is stamped at build time via -ldflags "-X main.version=...";
// see magefile.go's ldflags() and release.yml's VERSION export.
var version = "dev"

const asciiBanner = ` ____     ___    ____    ____            _____   _   _  __     __
| __ )   / _ \  |  _ \  / ___|          | ____| | \ | | \ \   / /
|  _ \  | | | | | |_) | \___ \   _____  |  _|   |  \| |  \ \ / /
| |_) | | |_| | |  __/   ___) | |_____| | |___  | |\  |   \ V /
|____/   \___/  |_|     |____/          |_____| |_| \_|    \_/`

func banner() {
	border := strings.Repeat("=", 60)
	fmt.Println(border)
	fmt.Println(asciiBanner)
	fmt.Println()
	fmt.Printf(" BlanketOps Environments CLI %s\n", version)
	fmt.Printf(" Connected to: %s\n", cmd.CurrentContext())
	fmt.Println(border)
}

// usageEntry pairs a command invocation with a one-line description of
// what it does, so `bops-env` with no args is self-documenting.
type usageEntry struct {
	cmd  string
	desc string
}

var usageEntries = [][]usageEntry{
	{
		{"bops-env version", "Print the installed bops-env build version"},
		{"bops-env install", "Install the BlanketOps Environments operator"},
		{"bops-env uninstall", "Remove the BlanketOps Environments operator"},
		{"bops-env dist", "Reserved (not yet implemented)"},
	},
	{
		{"bops-env self install", "Fetch and install the latest bops-env CLI binary"},
		{"bops-env self uninstall", "Remove a self-installed bops-env CLI binary"},
	},
	{
		{"bops-env dependencies list", "List individually addressable dependencies"},
		{"bops-env dependencies install [name]", "Install the platform stack, or just [name]"},
		{"bops-env dependencies uninstall [name]", "Remove the platform stack, or just [name]"},
		{"bops-env dependencies status [name]", "Show install status for all deps, or just [name]"},
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
	// version/--version is handled before the banner (which dials the
	// current kube context just to print it) and before cmd.Assets is
	// wired up — neither is needed just to report which build this is,
	// and it keeps `bops-env version` usable in scripts without banner
	// noise to strip out.
	if len(os.Args) >= 2 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println(version)
		return
	}

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
	// INSTALL / UNINSTALL — applies/removes the BlanketOps Environments
	// operator (CRDs, RBAC, controller-manager Deployment) published by
	// the environments-install repo. Upgrading the CLI binary itself is
	// not part of this — see the README's Installation section.
	// ---------------------------------------------------------------------
	case "install":
		if err := cmd.InstallOperator(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := cmd.UninstallOperator(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "dist":
		fmt.Println("ℹ️ dist command reserved (no-op for now)")
		return
	// ---------------------------------------------------------------------
	// SELF — fetches and installs/removes the bops-env CLI binary itself
	// from its own GitHub releases. Separate from INSTALL/UNINSTALL above,
	// which only ever touch the operator running on the cluster.
	// ---------------------------------------------------------------------
	case "self":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		switch os.Args[2] {
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
		default:
			fmt.Println("Unknown self command:", os.Args[2])
			usage()
			os.Exit(1)
		}
	// ---------------------------------------------------------------------
	// DEPENDENCIES — installs/removes the platform stack: every manifest
	// under the embedded dependencies/ tree.
	// ---------------------------------------------------------------------
	case "dependencies":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		depName := ""
		if len(os.Args) >= 4 {
			depName = os.Args[3]
		}
		switch os.Args[2] {
		case "install":
			var err error
			if depName == "" {
				err = cmd.InstallAll()
			} else {
				err = cmd.InstallDependency(depName)
			}
			if err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "uninstall":
			var err error
			if depName == "" {
				err = cmd.UninstallAll(uninstallPaths)
			} else {
				err = cmd.UninstallDependency(depName)
			}
			if err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "status":
			var err error
			if depName == "" {
				err = cmd.StatusAll()
			} else {
				err = cmd.StatusDependency(depName)
			}
			if err != nil {
				fmt.Println("❌", err)
				os.Exit(1)
			}
		case "list":
			cmd.ListDependencies()
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
