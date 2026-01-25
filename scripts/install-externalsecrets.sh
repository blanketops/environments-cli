#!/usr/bin/env bash
# install-external-secrets.sh
#
# Installs External Secrets Operator via Helm into a Kubernetes cluster.
#
# Usage:
#   NAMESPACE=external-secrets VERSION=0.10.7 ./install-external-secrets.sh
#
# Design goals:
# - Kubernetes 1.27+
# - Explicit CRD installation (avoids schema patch bugs)
# - Webhooks disabled (local / dev friendly)
# - Deterministic, repeatable installs

set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-external-secrets}"
RELEASE="${RELEASE:-external-secrets}"
REPO_NAME="${REPO_NAME:-external-secrets}"
REPO_URL="${REPO_URL:-https://charts.external-secrets.io}"
CHART="${CHART:-external-secrets}"

# 🔒 IMPORTANT:
# Known-good versions for Kubernetes 1.29+
# Override if you *know* what you’re doing.
VERSION="${VERSION:-0.10.7}"

TIMEOUT="${TIMEOUT:-5m}"

# -----------------------------------------------------------------------------
# Preconditions
# -----------------------------------------------------------------------------
command -v helm >/dev/null 2>&1 || {
  echo "❌ helm is required but not found"
  exit 1
}

command -v kubectl >/dev/null 2>&1 || {
  echo "❌ kubectl is required but not found"
  exit 1
}

# -----------------------------------------------------------------------------
# Helm repo setup
# -----------------------------------------------------------------------------
kubectl create ns external-secret
if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
  echo "➕ Adding helm repo $REPO_NAME -> $REPO_URL"
  helm repo add "$REPO_NAME" "$REPO_URL"
else
  echo "✅ Helm repo $REPO_NAME already present"
fi

echo "🔄 Updating helm repos..."
helm repo update

# -----------------------------------------------------------------------------
# Pre-clean (defensive)
# -----------------------------------------------------------------------------
echo "🧹 Ensuring no broken External Secrets CRDs exist..."
kubectl delete crd externalsecrets.external-secrets.io 2>/dev/null || true
kubectl delete crd secretstores.external-secrets.io 2>/dev/null || true
kubectl delete crd clustersecretstores.external-secrets.io 2>/dev/null || true

# -----------------------------------------------------------------------------
# Step 1: Install CRDs explicitly
# -----------------------------------------------------------------------------
echo "📦 Installing External Secrets CRDs (version: $VERSION)..."

helm template "$RELEASE" "$REPO_NAME/$CHART" \
  --version "$VERSION" \
  --include-crds \
  --set webhook.create=false \
  | kubectl apply -f -

# -----------------------------------------------------------------------------
# Step 2: Install / Upgrade controller (NO CRDs)
# -----------------------------------------------------------------------------
echo "🚀 Installing/upgrading External Secrets controller..."

helm upgrade --install "$RELEASE" "$REPO_NAME/$CHART" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --version "$VERSION" \
  --wait \
  --timeout "$TIMEOUT" \
  --set installCRDs=false \
  --set webhook.create=false

# -----------------------------------------------------------------------------
# Readiness checks
# -----------------------------------------------------------------------------
echo "⏳ Waiting for External Secrets controller to become ready..."
kubectl -n "$NAMESPACE" wait \
  --for=condition=Available \
  deployment/external-secrets \
  --timeout="$TIMEOUT"

# -----------------------------------------------------------------------------
# Post-install sanity checks
# -----------------------------------------------------------------------------
echo "🔍 Verifying webhook resources are NOT present..."

if kubectl get validatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
  echo "❌ ERROR: validating webhook unexpectedly present"
  exit 1
else
  echo "✅ No validating webhooks found"
fi

if kubectl get mutatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
  echo "❌ ERROR: mutating webhook unexpectedly present"
  exit 1
else
  echo "✅ No mutating webhooks found"
fi

# -----------------------------------------------------------------------------
# Final verification
# -----------------------------------------------------------------------------
echo "🔍 Verifying External Secrets CRDs..."
kubectl get crd | grep external-secrets

echo "✅ External Secrets installation completed successfully"
echo "   - Version:   $VERSION"
echo "   - Namespace: $NAMESPACE"
echo "   - Webhooks:  disabled"
