#!/usr/bin/env bash
# silence-finalizers.sh
# Force-remove finalizers from all BlanketOps-related CRs

set -euo pipefail

echo "🔕 Silencing finalizers across all namespaces..."

# Namespaces we touch
NAMESPACES=(
  default
  kapp-controller
  kapp-controller-packaging-global
  argo-events
  tekton-pipelines
  tekton-pipelines-resolvers
  shipwright-build
  crossplane-system
  external-secrets
)

# CRD groups we own or install transitively
CRD_GROUP_PATTERNS=(
  blanketops.dev
  shipwright.io
  tekton.dev
  argoproj.io
  external-secrets.io
  packaging.carvel.dev
  kappctrl.k14s.io
  sources.toolkit.fluxcd.io
  kustomize.toolkit.fluxcd.io
)

for pattern in "${CRD_GROUP_PATTERNS[@]}"; do
  echo "🔎 Processing CRDs matching *.$pattern"
  kubectl get crd -o name | grep "$pattern" || true
done

for ns in "${NAMESPACES[@]}"; do
  echo "📂 Namespace: $ns"
  for crd in $(kubectl get crd -o name); do
    kind=$(kubectl get "$crd" -o jsonpath='{.spec.names.kind}')
    plural=$(kubectl get "$crd" -o jsonpath='{.spec.names.plural}')
    group=$(kubectl get "$crd" -o jsonpath='{.spec.group}')

    # skip core k8s
    [[ "$group" == "core" ]] && continue

    kubectl get "$plural.$group" -n "$ns" --ignore-not-found -o name | while read -r obj; do
      echo "  🧨 Removing finalizers from $obj"
      kubectl patch "$obj" -n "$ns" --type=merge \
        -p '{"metadata":{"finalizers":[]}}' || true
    done
  done
done

echo "✅ Finalizers silenced"
