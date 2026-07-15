#!/usr/bin/env bash
# setup-knative-kourier.sh
# Points Knative Serving's networking layer at Kourier and waits for the
# net-kourier controller to be ready. Knative Serving doesn't pick a
# networking implementation on its own — config-network's ingress-class
# has to be set explicitly, same class as this repo's own Ingress
# resources use them.

set -euo pipefail

TIMEOUT="${TIMEOUT:-5m}"

command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required but not found"; exit 1; }

echo "[INFO] Waiting for net-kourier-controller rollout"
kubectl -n knative-serving rollout status deployment/net-kourier-controller --timeout="$TIMEOUT"

echo "[INFO] Setting Knative Serving's ingress-class to Kourier"
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'

echo "[INFO] Knative Serving is now using Kourier for ingress"
