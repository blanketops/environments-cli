# BlanketOps Environments — Platform Bootstrap CLI

A self-contained Kubernetes platform bootstrapper. A single binary installs a complete cloud-native delivery stack directly into a cluster using the Kubernetes API — **no kubectl required**.

The binary embeds all manifests and bootstrap scripts, making it ideal for minimal systems, immutable appliances, and automated cluster provisioning.

---

## Installation

### Linux
```bash
# amd64
curl -LO https://github.com/blanketops/environments-cli/releases/latest/download/bops-env-static
chmod +x bops-env-static
sudo mv bops-env-static /usr/local/bin/bops-env

# arm64
curl -LO https://github.com/blanketops/environments-cli/releases/latest/download/bops-env-static-arm64
chmod +x bops-env-static-arm64
sudo mv bops-env-static-arm64 /usr/local/bin/bops-env
```

### macOS / Build from source
```bash
git clone https://github.com/blanketops/environments-cli.git
cd environments-cli
mage install   # builds and installs to ~/.local/bin
```

---

## Usage

```bash
bops-env version                # installed CLI version
bops-env install                # install the operator
bops-env uninstall              # uninstall the operator
bops-env self install           # update the CLI binary
bops-env self uninstall         # uninstall the CLI binary
bops-env dependencies install   # install platform stack
bops-env dependencies uninstall <name>  # uninstall a dependency
bops-env cluster up/down/status [name]  # cluster management
```

---

## Secrets Bootstrap

Bootstrap the canonical BlanketOps secret set (Git SSH keys, registry auth, GitHub tokens, Crossplane token) into an external secret store.

```bash
bops-env secrets bootstrap <provider> [args...] [flags]
```

### Providers

| Provider | Arguments | Example |
|----------|-----------|---------|
| **aws** | `<region> [path-prefix]` | `bops-env secrets bootstrap aws us-east-1` |
| **azure** | `<vault-url>` | `bops-env secrets bootstrap azure https://myvault.vault.azure.net` |
| **gcp** | `<project-id>` | `bops-env secrets bootstrap gcp my-project` |
| **vault** | `<addr> <token> [mount]` | `bops-env secrets bootstrap vault http://vault:8200 root secret` |

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run`, `--preview` | Show what operations would be performed without executing |
| `-y`, `--yes` | Skip confirmation prompt (for automation) |

### Secrets sourced from environment

| Secret | Environment Variable | Fallback |
|--------|---------------------|----------|
| Git SSH private key | `BOPS_SSH_PRIVATE_KEY_FILE` | `~/.ssh/id_ed25519` |
| Git SSH public key | `BOPS_SSH_PUBLIC_KEY_FILE` | `~/.ssh/id_ed25519.pub` |
| Registry config | `BOPS_REGISTRY_CONFIG` or `BOPS_GHCR_USER` + `BOPS_GHCR_TOKEN` | placeholder |
| GitHub webhook secret | `BOPS_GITHUB_WEBHOOK_SECRET` | placeholder |
| GitHub API token | `BOPS_GITHUB_API_TOKEN` | placeholder |
| Crossplane GitHub token | `BOPS_CROSSPLANE_GITHUB_TOKEN` | placeholder |

Placeholders are skipped (not written) so a dry run never clobbers existing real values.

### Examples

```bash
# Interactive — shows preview, asks to proceed
bops-env secrets bootstrap aws us-east-1

# Dry run — preview only, no changes
bops-env secrets bootstrap aws us-east-1 --dry-run
bops-env secrets bootstrap gcp my-project --preview

# Automation — skip confirmation
bops-env secrets bootstrap azure https://myvault.vault.azure.net -y
bops-env secrets bootstrap vault http://vault:8200 token secret --yes
```

### Verify

```bash
bops-env secrets verify <provider> [args...]
```

Lists existing BlanketOps secrets in the provider to confirm they exist.

---

## Design Goals
- Air-gapped clusters, bare metal / edge nodes
- Immutable appliances / Gokrazy
- Ephemeral CI environments

---

## Verification

Every release is signed and attested:
- **Signed:** Binaries signed with `cosign` (keyless mode)
- **Attested:** SLSA-compliant provenance attestation

Verify after download:
```bash
curl -LO https://github.com/blanketops/environments-cli/releases/latest/download/bops-env-static.sig
cosign verify-blob --certificate-identity-regexp ".*" --signature bops-env-static.sig bops-env-static
gh attest verify bops-env-static --owner blanketops
```