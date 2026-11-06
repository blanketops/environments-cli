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