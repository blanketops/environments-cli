#!/usr/bin/env bash
# install-crossplane.sh
# Installs Crossplane via Helm

set -euo pipefail

NAMESPACE="${NAMESPACE:-crossplane-system}"
RELEASE="${RELEASE:-crossplane}"
REPO_NAME="crossplane-stable"
REPO_URL="https://charts.crossplane.io/stable"
CHART="crossplane-stable/crossplane"
VERSION="${VERSION:-}"
TIMEOUT="${TIMEOUT:-5m}"

# never allow VERSION=latest
if [[ "${VERSION}" == "latest" ]]; then
  VERSION=""
fi

command -v helm >/dev/null 2>&1 || { echo "helm is required but not found"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required but not found"; exit 1; }

echo "Preparing Helm repo..."

# reset repo to avoid stale index problems
helm repo remove "$REPO_NAME" >/dev/null 2>&1 || true
helm repo add "$REPO_NAME" "$REPO_URL"

echo "Updating Helm repositories..."
helm repo update >/dev/null

echo "Installing/upgrading Crossplane..."

if [[ -n "$VERSION" ]]; then
  helm upgrade --install "$RELEASE" "$CHART" \
    --namespace "$NAMESPACE" \
    --create-namespace \
    --version "$VERSION" \
    --wait \
    --timeout "$TIMEOUT"
else
  helm upgrade --install "$RELEASE" "$CHART" \
    --namespace "$NAMESPACE" \
    --create-namespace \
    --wait \
    --timeout "$TIMEOUT"
fi

echo "Waiting for Crossplane deployment..."

kubectl -n "$NAMESPACE" rollout status deployment/crossplane --timeout="$TIMEOUT"

echo "Crossplane installation completed."