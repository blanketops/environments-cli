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
	"context"
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
var version = "latest"

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
	{
		{"bops-env secrets bootstrap <provider> [args...]", "Bootstrap BlanketOps secrets into a secret store (aws, azure, gcp, vault)"},
		{"bops-env secrets verify <provider> [args...]", "Verify bootstrapped secrets exist in a secret store"},
	},
	{
		{"bops-env report [flags]", "Send test/benchmark results to Allure server"},
		{"bops-env report-server [flags]", "Start report receiver server that forwards to Allure"},
	},
}

const (
	helpInstall = `Install the BlanketOps Environments operator (CRDs, RBAC, controller-manager).

Usage:
  bops-env install

This applies the operator manifests from the environments-install repository.
The operator manages Environment custom resources and reconciles builds, deployments,
and secrets.

Examples:
  bops-env install
`

	helpUninstall = `Remove the BlanketOps Environments operator.

Usage:
  bops-env uninstall

This removes the operator Deployment, CRDs, RBAC, and associated resources.

Examples:
  bops-env uninstall
`

	helpSelf = `Manage the bops-env CLI binary itself.

Usage:
  bops-env self <command>

Commands:
  install    Fetch and install the latest bops-env CLI binary from GitHub releases
  uninstall  Remove a self-installed bops-env CLI binary

Examples:
  bops-env self install
  bops-env self uninstall
`

	helpDependencies = `Manage the platform stack dependencies (Carvel, Argo Events, Tekton, Shipwright, etc.).

Usage:
  bops-env dependencies <command> [name]

Commands:
  list       List all individually addressable dependencies
  install    Install the full platform stack, or a specific dependency by name
  uninstall  Remove the full platform stack, or a specific dependency by name
  status     Show install status for all dependencies, or just [name]

Dependency names (use with install/uninstall/status):
  carvel, argoevents, tekton, tekton-dashboard, shipwright, crossplane,
  crossplane-provider, externalsecrets, vault, flux, flux-crds, flux-ui,
  buildpacks, kaniko, buildah, knative, kourier, knative-kourier-setup

Examples:
  bops-env dependencies list
  bops-env dependencies install
  bops-env dependencies install tekton
  bops-env dependencies status flux
  bops-env dependencies uninstall vault
`

	helpCluster = `Manage kind clusters for local development.

Usage:
  bops-env cluster <command> [name]

Commands:
  up      Create the named cluster if it doesn't exist
  down    Delete the named cluster
  status  Show whether the named cluster exists

Examples:
  bops-env cluster up
  bops-env cluster up my-cluster
  bops-env cluster status
  bops-env cluster down
`

	helpSecrets = `Bootstrap and verify BlanketOps secrets in external secret stores.

Usage:
  bops-env secrets <command> <provider> [args...]

Commands:
  bootstrap  Create all required BlanketOps secrets in the provider
  verify     List and verify existing BlanketOps secrets in the provider

Providers:
  aws     AWS Secrets Manager
          Args: <region> [path-prefix]
          Example: bops-env secrets bootstrap aws us-east-1 /blanketops
  
  azure     Azure Key Vault
          Args: <vault-url>
          Example: bops-env secrets bootstrap azure https://myvault.vault.azure.net
  
  gcp       Google Cloud Secret Manager
          Args: <project-id>
          Example: bops-env secrets bootstrap gcp my-gcp-project
  
  vault     HashiCorp Vault (KV v2)
          Args: <addr> <token> [mount]
          Example: bops-env secrets bootstrap vault http://vault:8200 root secret

Examples:
  bops-env secrets bootstrap aws us-east-1
  bops-env secrets verify gcp my-project
  bops-env secrets bootstrap vault http://localhost:8200 mytoken secret
`

	helpReport = `Send test/benchmark results to Allure server.

Usage:
  bops-env report [flags]

Flags:
  --server-url     Allure server URL (ALLURE_SERVER_URL)
  --project-name   Allure project name (ALLURE_PROJECT_NAME)
  --api-key        API key for Allure server (ALLURE_API_KEY)
  --file           Input file (default: stdin)
  --run-id         Run ID (default: timestamp)
  --type           Test type: test, benchmark, scenario

Examples:
  # Send test results from file
  bops-env report -f test-output.json
  
  # Send benchmark results from stdin
  go test ./... -bench=. -benchmem -json | bops-env report --type benchmark
  
  # Send with custom run ID
  bops-env report -f output.json --run-id "ci-123"

Environment variables:
  ALLURE_SERVER_URL    Base URL of Allure server (e.g. https://allure.blanketops.io)
  ALLURE_PROJECT_NAME  Project name in Allure
  ALLURE_API_KEY       API key for authentication
`

	helpReportServer = `Start a report receiver server that forwards to Allure.

Usage:
  bops-env report-server [flags]

Flags:
  --port       Server port (default: 8080)
  --allure-url Allure server URL (ALLURE_SERVER_URL)

Environment variables:
  ALLURE_SERVER_URL    Base URL of Allure server
  ALLURE_API_KEY       API key for Allure server
  ALLURE_PROJECT_NAME  Project name in Allure
`
)

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

func printHelpAndExit(helpText string) {
	fmt.Println(helpText)
	os.Exit(0)
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
		// Check for help flag
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpInstall)
			}
		}
		if err := cmd.InstallOperator(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "uninstall":
		// Check for help flag
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpUninstall)
			}
		}
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
		// Check for help flag first (before subcommand)
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpSelf)
			}
		}
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
		// Check for help flag first (before subcommand)
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpDependencies)
			}
		}
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
		// Check for help flag first (before subcommand)
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpCluster)
			}
		}
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
	// ---------------------------------------------------------------------
	// SECRETS — bootstrap/verify secrets in external secret stores
	// ---------------------------------------------------------------------
	case "secrets":
		ctx := context.Background()
		// Check for help flag first (before subcommand)
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				printHelpAndExit(helpSecrets)
			}
		}
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		subCmd := os.Args[2]
		provider := ""
		args := []string{}
		if len(os.Args) >= 4 {
			provider = os.Args[3]
		}
		if len(os.Args) >= 5 {
			args = os.Args[4:]
		}
		var err error
		switch subCmd {
		case "bootstrap":
			err = cmd.SecretsBootstrap(ctx, provider, args)
		case "verify":
			err = cmd.SecretsVerify(ctx, provider, args)
		default:
			fmt.Println("Unknown secrets command:", subCmd)
			usage()
			os.Exit(1)
		}
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	// ---------------------------------------------------------------------
	// REPORT — send test/benchmark results to Allure
	// ---------------------------------------------------------------------
	case "report":
		if len(os.Args) >= 3 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
			printHelpAndExit(helpReport)
		}
		if err := cmd.ReportCommand().Execute(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	case "report-server":
		if len(os.Args) >= 3 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
			printHelpAndExit(helpReportServer)
		}
		if err := cmd.ReportServerCommand().Execute(); err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		usage()
		os.Exit(1)
	}
}
