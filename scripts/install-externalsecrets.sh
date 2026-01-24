#!/usr/bin/env bash
# install-external-secrets.sh
#
# Installs External Secrets Operator via Helm into a Kubernetes cluster.
#
# Usage:
#   NAMESPACE=external-secrets VERSION=0.10.5 ./install-external-secrets.sh
#
# Notes:
# - This script assumes NO pre-existing External Secrets CRDs.
# - Admission webhooks are DISABLED intentionally (dev / local clusters).
# - This avoids TLS + cert-manager + webhook race conditions.

set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-external-secrets}"
RELEASE="${RELEASE:-external-secrets}"
REPO_NAME="${REPO_NAME:-external-secrets}"
REPO_URL="${REPO_URL:-https://charts.external-secrets.io}"
CHART="${CHART:-external-secrets}"
VERSION="${VERSION:-}"     # optional, e.g. "0.10.5"
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
if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
  echo "➕ Adding helm repo $REPO_NAME -> $REPO_URL"
  helm repo add "$REPO_NAME" "$REPO_URL"
else
  echo "✅ Helm repo $REPO_NAME already present"
fi

echo "🔄 Updating helm repos..."
helm repo update

# -----------------------------------------------------------------------------
# Install / Upgrade External Secrets
# -----------------------------------------------------------------------------
echo "🚀 Installing/upgrading External Secrets (release: $RELEASE) into namespace $NAMESPACE..."

upgrade_args=(
  upgrade --install "$RELEASE" "$REPO_NAME/$CHART"
  --namespace "$NAMESPACE"
  --create-namespace
  --wait
  --timeout "$TIMEOUT"

  # ---------------------------------------------------------
  # 🔥 CRITICAL FIX: Disable admission webhooks
  # ---------------------------------------------------------
  --set webhook.create=false
)

if [ -n "$VERSION" ]; then
  upgrade_args+=(--version "$VERSION")
fi

helm "${upgrade_args[@]}"

# -----------------------------------------------------------------------------
# Readiness checks
# -----------------------------------------------------------------------------
# NOTE:
# - With webhook.create=false:
#   - external-secrets-webhook DOES NOT exist
#   - external-secrets-cert-controller DOES NOT exist
# - Only the main controller must be ready
# -----------------------------------------------------------------------------

echo "⏳ Waiting for External Secrets controller to become ready..."
kubectl -n "$NAMESPACE" wait \
  --for=condition=Available \
  deployment/external-secrets \
  --timeout="$TIMEOUT"

# -----------------------------------------------------------------------------
# Post-install sanity checks
# -----------------------------------------------------------------------------
echo "🔍 Verifying no External Secrets webhooks are registered..."
if kubectl get validatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
  echo "⚠️  WARNING: External Secrets webhook still present"
else
  echo "✅ No External Secrets validating webhooks found"
fi

if kubectl get mutatingwebhookconfiguration 2>/dev/null | grep -q external-secrets; then
  echo "⚠️  WARNING: External Secrets webhook still present"
else
  echo "✅ No External Secrets mutating webhooks found"
fi

echo "✅ External Secrets installation completed successfully (webhook disabled)."
