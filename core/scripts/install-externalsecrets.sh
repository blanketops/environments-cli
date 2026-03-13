#!/usr/bin/env bash

#

# install-external-secrets.sh

#

# Installs External Secrets Operator via Helm.

# Automatically resolves latest chart version.

# Webhooks intentionally disabled for dev clusters.

set -euo pipefail

# -----------------------------------------------------------------------------

# Configuration

# -----------------------------------------------------------------------------

NAMESPACE="${NAMESPACE:-external-secrets}"
RELEASE="${RELEASE:-external-secrets}"
REPO_NAME="${REPO_NAME:-external-secrets}"
REPO_URL="${REPO_URL:-https://charts.external-secrets.io}"
CHART="${CHART:-external-secrets}"
TIMEOUT="${TIMEOUT:-5m}"

# -----------------------------------------------------------------------------

# Preconditions

# -----------------------------------------------------------------------------

require() {
command -v "$1" >/dev/null 2>&1 || {
echo "❌ $1 is required but not installed"
exit 1
}
}

require helm
require kubectl
require jq

# -----------------------------------------------------------------------------

# Helm repo setup

# -----------------------------------------------------------------------------

if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
echo "➕ Adding Helm repo $REPO_NAME -> $REPO_URL"
helm repo add "$REPO_NAME" "$REPO_URL"
else
echo "✅ Helm repo $REPO_NAME already present"
fi

echo "🔄 Updating Helm repos..."
helm repo update >/dev/null

# -----------------------------------------------------------------------------

# Resolve latest chart version

# -----------------------------------------------------------------------------

echo "🔎 Resolving latest External Secrets chart version..."

VERSION=$(
helm search repo "$REPO_NAME/$CHART" -o json | jq -r '.[0].version'
)

if [[ -z "$VERSION" || "$VERSION" == "null" ]]; then
echo "❌ Unable to resolve External Secrets chart version"
exit 1
fi

echo "📦 Latest chart version: $VERSION"

# -----------------------------------------------------------------------------

# Install / Upgrade External Secrets

# -----------------------------------------------------------------------------

echo "🚀 Installing External Secrets (release: $RELEASE)"

helm upgrade --install "$RELEASE" "$REPO_NAME/$CHART" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --wait \
  --timeout "$TIMEOUT" \
  --version "$VERSION" \
#  --set webhook.create=false

# -----------------------------------------------------------------------------

# Wait for CRDs

# -----------------------------------------------------------------------------

echo "⏳ Waiting for External Secrets CRDs..."

kubectl wait 
--for=condition=Established 
crd/externalsecrets.external-secrets.io 
--timeout="$TIMEOUT"

# -----------------------------------------------------------------------------

# Wait for controller

# -----------------------------------------------------------------------------

echo "⏳ Waiting for External Secrets controller..."

kubectl -n "$NAMESPACE" wait 
--for=condition=Available 
deployment/external-secrets 
--timeout="$TIMEOUT"

# -----------------------------------------------------------------------------

# Post-install verification

# -----------------------------------------------------------------------------

echo "🔍 Verifying installation..."

kubectl get pods -n "$NAMESPACE"

echo "🔍 Checking CRDs..."

kubectl get crd | grep external-secrets || true

echo "🔍 Checking webhook absence..."

if kubectl get validatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
echo "⚠️  WARNING: External Secrets validating webhook detected"
else
echo "✅ No validating webhooks found"
fi

if kubectl get mutatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
echo "⚠️  WARNING: External Secrets mutating webhook detected"
else
echo "✅ No mutating webhooks found"
fi

echo "🎉 External Secrets installed successfully"
echo "Namespace: $NAMESPACE"
echo "Release:   $RELEASE"
echo "Version:   $VERSION"
