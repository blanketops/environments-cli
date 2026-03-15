# 🚀 BlanketOps Environments — CLI (v0.0.1)

BlanketOps `Environments` CLI is a self-contained, zero-dependency Kubernetes bootstrapper.
One binary installs your entire software delivery environment:

It provides a fast, repeatable way to bring up a complete software delivery environment for BlanketOps — from a clean machine to a working Kubernetes platform — using a single binary.

The CLI is intentionally explicit, filesystem-driven, and easy to reason about.

## 🎯 What this tool achieves

BlanketOps Environments is designed to:

- ### Lower the cost of onboarding
  - New contributors get a working environment quickly
  - No tribal knowledge required

- ### Provide a consistent baseline
  - Same dependencies
  - Same order
  - Same behavior across machines

- ### Make infrastructure visible
  - Manifests live on disk
  - Scripts are readable
  - Failures are observable

- ### Support fast iteration
- Easy to reset
- Easy to debug
- Easy to extend

This tool focuses on environment bootstrapping, not long-term cluster management.

## 🧱 Platform components installed

When you run:

```sh
blanketops-environments install
```

The following components are installed into the cluster.

### Core delivery platform

- Carvel (kapp)
- Argo Events
- Tekton Pipelines
- Tekton Dashboard
- Shipwright Build

### Control plane & GitOps

- Flux CD (CRDs + UI)
- Crossplane
- External Secrets

### Build strategies

- Buildpacks v3
- Kaniko
- Buildah (Shipwright-managed)

All manifests are sourced from:

```tree
dependencies/
```

and applied deterministically.

## 🖥 CLI Usage

### Primary commands (stable)

```sh
blanketops-environments install
blanketops-environments uninstall
blanketops-environments dist
```

These commands are considered part of the stable CLI contract.

### Explicit dependency aliases

```sh
blanketops-environments dependencies install
blanketops-environments dependencies uninstall
```

These are aliases of install and uninstall, provided to make intent explicit.

### Cluster lifecycle (Kind, MVP scope)

```sh
blanketops-environments cluster up
blanketops-environments cluster down
blanketops-environments cluster status
```

Cluster behavior in v0.0.1:

- Uses Kind as the cluster backend
- Creates a cluster if one does not exist
- Waits for the Kubernetes API and node readiness
- Fails fast if prerequisites are missing

## 🛠 Installation

### Build the CLI

```sh
make build
```

### Install locally

```sh
make install
```

The binary is installed to:

- `$HOME/.local/bin` (preferred)
- `$HOME/bin` (fallback)

Ensure the location is on your PATH.

## 🚀 Recommended workflow

```sh
# Create a local cluster
blanketops-environments cluster up

# Install platform dependencies
blanketops-environments install

# Verify
kubectl get pods -A
```

To reset everything:

```sh
blanketops-environments uninstall
blanketops-environments cluster down
```

## 📂 Project structure (current)

```tree
cli/
├── main.go                # CLI entrypoint
├── cmd/                   # Command wiring
│ ├── install.go
│ ├── uninstall.go
│ └── dist.go
│
├── core/                  # Authoritative system logic
│ ├── cluster.go           # Cluster lifecycle (Kind)
│ ├── dependencies.go      # Dependency installation
│ ├── kube.go              # client-go setup
│ ├── apply.go             # Kubernetes apply engine
│ ├── uninstall.go         # Safe teardown logic
│ └── wait.go              # Readiness checks
│
├── dependencies/          # Filesystem manifests
│ ├── tekton/
│ ├── shipwright/
│ ├── flux/
│ ├── crossplane/
│ └── cluster_strategies/
│
├── scripts/               # Shell helpers
├── util/                  # Low-level helpers
├── Makefile
└── README.md
```

## 🧪 Testing status

- Manual testing via Kind
- Installer behavior verified end-to-end
- Automated tests planned post–v0.0.1

The focus in this release is correct sequencing and observability.

## 🧭 Design principles

- CLI is the entry point
- `core/` owns behavior
- Manifests are explicit
- State changes are reversible
- Errors should be visible

This project favors clarity over cleverness.

## 🤝 Contributing

The project is early-stage but stabilizing.

- Issues and feedback are welcome
- Pull requests are welcome
- Architectural discussions are encouraged

If something is unclear, that’s a signal to improve the tool.

## 🧠 Closing note

BlanketOps Environments exists to make Kubernetes environments `repeatable`, `understandable`, and `disposable`.

If you can:

- inspect what was applied
- understand why something failed
- tear everything down and start again

then the tool is doing its job.

```

```
