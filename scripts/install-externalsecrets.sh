#!/usr/bin/env bash
# install-external-secrets.sh
#
# Deterministic install of External Secrets Operator
# - Kubernetes 1.27+
# - Explicit CRD install
# - Webhooks DISABLED (local/dev safe)
# - Idempotent, repeatable

set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-external-secrets}"
RELEASE="${RELEASE:-external-secrets}"
REPO_NAME="${REPO_NAME:-external-secrets}"
REPO_URL="${REPO_URL:-https://charts.external-secrets.io}"
CHART="${CHART:-external-secrets}"
VERSION="${VERSION:-0.10.7}"
TIMEOUT="${TIMEOUT:-5m}"

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
# Helm repo
# -----------------------------------------------------------------------------
if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
  helm repo add "$REPO_NAME" "$REPO_URL"
fi
helm repo update

# -----------------------------------------------------------------------------
# Defensive CRD cleanup (prevents storedVersions bugs)
# -----------------------------------------------------------------------------
echo "🧹 Removing any existing External Secrets CRDs..."
kubectl get crd | awk '/external-secrets.io/ {print $1}' | xargs -r kubectl delete crd

# -----------------------------------------------------------------------------
# Install CRDs explicitly (NO controller, NO webhooks)
# -----------------------------------------------------------------------------
echo "📦 Installing External Secrets CRDs ($VERSION)..."
helm template "$RELEASE" "$REPO_NAME/$CHART" \
  --version "$VERSION" \
  --include-crds \
  --set installCRDs=false \
  --set webhook.create=false \
  --set certController.create=false \
  | kubectl apply -f -

# -----------------------------------------------------------------------------
# Install controller ONLY (no CRDs, no webhooks)
# -----------------------------------------------------------------------------
echo "🚀 Installing External Secrets controller..."
helm upgrade --install "$RELEASE" "$REPO_NAME/$CHART" \
  --namespace "$NAMESPACE" \
  --version "$VERSION" \
  --wait \
  --timeout "$TIMEOUT" \
  --set installCRDs=false \
  --set webhook.create=false \
  --set certController.create=false

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
echo "🔍 Verifying no ESO webhooks exist..."
kubectl get validatingwebhookconfiguration | grep external-secrets && exit 1 || true
kubectl get mutatingwebhookconfiguration | grep external-secrets && exit 1 || true

echo "🔍 Verifying CRDs..."
kubectl get crd | grep external-secrets

echo "✅ External Secrets installed successfully"
echo "   Version:   $VERSION"
echo "   Namespace: $NAMESPACE"
echo "   Webhooks:  disabled"
