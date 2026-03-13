#!/usr/bin/env bash
# install-crossplane.sh
# Installs Crossplane via Helm into a Kubernetes cluster.
# Usage:
#   NAMESPACE=custom-name VERSION=1.8.0 ./install-crossplane.sh

set -euo pipefail

NAMESPACE="${NAMESPACE:-crossplane-system}"
RELEASE="${RELEASE:-crossplane}"
REPO_NAME="${REPO_NAME:-crossplane-stable}"
REPO_URL="${REPO_URL:-https://charts.crossplane.io/stable}"
CHART="${CHART:-crossplane}"
VERSION="${VERSION:-}"   # optional, e.g. "1.9.0"
TIMEOUT="${TIMEOUT:-5m}"

command -v helm >/dev/null 2>&1 || { echo "helm is required but not found"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required but not found"; exit 1; }

# add repo if missing
if ! helm repo list | awk '{print $1}' | grep -qx "$REPO_NAME"; then
    echo "Adding helm repo $REPO_NAME -> $REPO_URL"
    helm repo add "$REPO_NAME" "$REPO_URL"
else
    echo "Helm repo $REPO_NAME already present"
fi

echo "Updating helm repos..."
helm repo update

set +e
echo "Installing/upgrading Crossplane (release: $RELEASE) into namespace $NAMESPACE..."
upgrade_args=(upgrade --install "$RELEASE" "$REPO_NAME/$CHART" --namespace "$NAMESPACE" --create-namespace --wait --timeout "$TIMEOUT")
if [ -n "$VERSION" ]; then
    upgrade_args+=(--version "$VERSION")
fi
set -e
helm "${upgrade_args[@]}"

# wait for Crossplane deployment to be ready (best-effort)
echo "Waiting for Crossplane deployment to become ready..."
if kubectl -n "$NAMESPACE" wait --for=condition=Available deployment/crossplane --timeout="$TIMEOUT"; then
    echo "Crossplane is ready in namespace $NAMESPACE"
else
    echo "Timed out waiting for deployment/crossplane to be Available. Check 'kubectl -n $NAMESPACE get pods' and 'helm status $RELEASE'"
    exit 1
fi

echo "Crossplane installation completed."