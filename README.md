# 🚀 BlanketOps Environments — MVP Bootstrapper

BlanketOps CLI is a self-contained, zero-dependency Kubernetes bootstrapper.
One binary installs your entire software delivery environment:

- Calico CNI
- Tekton Pipelines
- Tekton Dashboard
- Shipwright Build (with automatic webhook certificate setup)
- Build Strategies (Buildpacks, Buildah, Kaniko)
- FluxCD Core + UI

```
💡 All manifests and scripts are embedded directly in the binary. No kubectl, no filesystem dependencies, no internet required.
```

## Perfect for:

- Air-gapped bootstrapping
- Gokrazy deployments
- Bare metal appliances
- Minimalist K8s nodes
- Ephemeral environments

# ✨ Features

## 🔧 Pure Go Kubernetes Apply Engine

- No kubectl.
- No exec.Command.
- Everything goes through:
  - client-go dynamic client
  - discovery-backed REST mapper
  - unstructured resource decoding

CRD → wait → resource apply sequence

## 📦 Fully Embedded Manifests

The binary contains:

```
- manifests/calico/\_
- manifests/tekton/\_
- manifests/shipwright/\_
- manifests/flux/\_
- manifests/cluster*strategies/*
- scripts/\_
```

Nothing is read from disk.

## 🔐 Shipwright Webhook Certificate Automation

The installer:

- Generates CSR
- Approves it
- Creates TLS secret
- Patches CRDs with CA bundle
- Restarts webhook
- Waits for deployment readiness

Done automatically.

## 🌩️ Gokrazy-Ready Static Build

Run:

```
make static
```

→ produces a fully static Linux/amd64 binary that drops directly into a gokrazy image.

## 🛠️ Installation

Build:

```
make build
```

Install to $HOME/.local/bin or fallback $HOME/bin:

```
make install
```

Static build (gokrazy):

```

make static

```

## 🚀 Usage

Install the entire platform stack:

```

blanketops-environments -install

```

This deploys:

- Calico
- Flux CD CRDs
- Tekton Pipelines
- Tekton Dashboard
- Shipwright + certs
- Build Strategies
- Flux CD UI configuration

Uninstall everything:

```

blanketops-environments -uninstall

```

List embedded manifest groups:

```

blanketops-environments -list

```

## 📂 Project Structure

```

blanketops/
│
├── main.go # CLI entrypoint
├── install.go # Install pipeline
├── uninstall.go # Uninstall logic
├── apply.go # Generic YAML apply engine
├── util.go # K8s client helpers
├── util_go.go # FS + embed helpers
│
├── manifests/ # Embedded manifests
│ ├── calico/
│ ├── tekton/
│ ├── flux/
│ ├── shipwright/
│ └── cluster_strategies/
│
├── scripts/
│ └── setup-shipwright-cert.sh
│
├── manifests_embed.go # go:embed configuration
├── Makefile # build, install, static build
└── README.md

```

# 🧪 Local Testing

```

kind create cluster
blanketops-environments -install

```

Verify core components:

```

kubectl get pods -A

```

⚙️ Gokrazy Workflow

Build static binary:

```
make static
```

Add binary to a gokrazy package:

```
gok add ./bin/blanketops-environments-static
```

Rebuild image:

```
gok build
```

Boot via QEMU or hardware.

Once inside gokrazy:

```
/user/blanketops-environments -install
```

🤝 Contributing

Pull requests welcome.
This repository is still early-stage but stabilizing fast.

```

```
