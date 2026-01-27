#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-external-secrets}"
RELEASE="${RELEASE:-external-secrets}"
REPO_NAME="${REPO_NAME:-external-secrets}"
REPO_URL="${REPO_URL:-https://charts.external-secrets.io}"
VERSION="${VERSION:-0.10.7}"
TIMEOUT="${TIMEOUT:-5m}"

CRD_URL="https://raw.githubusercontent.com/external-secrets/external-secrets/v${VERSION}/deploy/crds/bundle.yaml"

# -----------------------------------------------------------------------------
# Preconditions
# -----------------------------------------------------------------------------
command -v helm >/dev/null 2>&1 || { echo "❌ helm not found"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "❌ kubectl not found"; exit 1; }

# -----------------------------------------------------------------------------
# Namespace
# -----------------------------------------------------------------------------
kubectl get ns "$NAMESPACE" >/dev/null 2>&1 || kubectl create ns "$NAMESPACE"

# -----------------------------------------------------------------------------
# Helm repo (MUST exist)
# -----------------------------------------------------------------------------
if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
  helm repo add "$REPO_NAME" "$REPO_URL"
fi
helm repo update

# -----------------------------------------------------------------------------
# Install CRDs (authoritative source for 0.10.x)
# -----------------------------------------------------------------------------
echo "📦 Installing External Secrets CRDs ($VERSION)..."
kubectl apply -f "$CRD_URL"

# -----------------------------------------------------------------------------
# Install / Upgrade controller (Helm OWNS runtime resources)
# -----------------------------------------------------------------------------
echo "🚀 Installing External Secrets controller..."
helm upgrade --install "$RELEASE" "$REPO_NAME/external-secrets" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --version "$VERSION" \
  --wait \
  --timeout "$TIMEOUT" \
  --set installCRDs=false

# -----------------------------------------------------------------------------
# Readiness
# -----------------------------------------------------------------------------
kubectl -n "$NAMESPACE" wait \
  --for=condition=Available \
  deployment/external-secrets \
  --timeout="$TIMEOUT"

# -----------------------------------------------------------------------------
# Sanity checks
# -----------------------------------------------------------------------------
echo "🔍 Verifying ExternalSecret CRD exists..."
kubectl get crd externalsecrets.external-secrets.io

echo "🔍 Pods:"
kubectl get pods -n "$NAMESPACE"

echo "✅ External Secrets installed successfully"
echo "   Version:   $VERSION"
echo "   Namespace: $NAMESPACE"
