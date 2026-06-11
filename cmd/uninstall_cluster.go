package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

// ---------------------------------------------------------------------------
// RunUninstallScripts
// ---------------------------------------------------------------------------
func RunUninstallScripts() error {
	_ = exec.Command("helm", "uninstall", "crossplane", "-n", "crossplane-system").Run()
	_ = exec.Command("helm", "uninstall", "external-secrets", "-n", "external-secrets").Run()
	_ = exec.Command("helm", "uninstall", "flux", "-n", "flux-system").Run()

	if err := run("scripts/silence-all-finalizers.sh"); err != nil {
		return err
	}

	_ = run("scripts/install-crossplane.sh", "--uninstall")
	_ = run("scripts/install-externalsecrets.sh", "--uninstall")

	return nil
}

// ---------------------------------------------------------------------------
// Cluster-wide cleanup
// ---------------------------------------------------------------------------
func UninstallClusterResources() error {
	if err := deleteRBAC(); err != nil {
		return err
	}

	if err := deleteCRDs(); err != nil {
		return err
	}

	return deleteNamespaces()
}

func run(script string, args ...string) error {
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deleteRBAC() error {
	fmt.Println("🔐 Removing RBAC resources")

	cmd := exec.Command("bash", "-c", `
set -e

echo "🧹 Deleting BlanketOps RBAC"
kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/managed-by=blanketops \
  --ignore-not-found

echo "🧹 Deleting Crossplane RBAC (label-based)"
kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/part-of=crossplane \
  --ignore-not-found

kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/name=crossplane \
  --ignore-not-found

echo "🧹 Deleting Argo Events RBAC"
kubectl delete clusterrole,clusterrolebinding \
  argo-events-role \
  argo-events-binding \
  argo-events-aggregate-to-admin \
  argo-events-aggregate-to-edit \
  argo-events-aggregate-to-view \
  --ignore-not-found

echo "🧹 Deleting Crossplane RBAC (name-based fallback)"
kubectl get clusterrole -o name | grep -E 'crossplane|rbac-manager|provider-' | xargs -r kubectl delete --ignore-not-found
kubectl get clusterrolebinding -o name | grep -E 'crossplane|rbac-manager|provider-' | xargs -r kubectl delete --ignore-not-found

echo "🧹 Deleting External Secrets RBAC"
kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/name=external-secrets \
  --ignore-not-found

echo "🧹 Deleting Flux RBAC"
kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/part-of=flux \
  --ignore-not-found

echo "🧹 Deleting Argo Events RBAC"
kubectl delete clusterrole,clusterrolebinding \
  -l app.kubernetes.io/part-of=argo-events \
  --ignore-not-found

echo "✅ RBAC cleanup complete"
`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deleteCRDs() error {
	fmt.Println("🧨 Removing ALL CustomResourceDefinitions (authoritative)")

	cmd := exec.Command("bash", "-c", `
set -e

echo "🔕 Removing finalizers from all CRDs"
kubectl get crd -o name | while read crd; do
  kubectl patch "$crd" --type=merge -p '{"metadata":{"finalizers":[]}}' || true
done

echo "🔥 Deleting all CRDs"
kubectl get crd -o name | xargs -r kubectl delete --ignore-not-found

echo "✅ All CRDs deleted"
`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// func runScript(script string, args ...string) error {
// 	fmt.Printf("⚙️  Running %s %v\n", script, args)

// 	cmd := exec.Command("bash", append([]string{script}, args...)...)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	return cmd.Run()
// }

func deleteNamespaces() error {
	fmt.Println("🧹 Removing system namespaces")

	namespaces := []string{
		"flux-system",
		"external-secrets",
		"crossplane-system",
		"argo-events",
		"tekton-pipelines",
		"tekton-pipelines-resolvers",
		"shipwright-build",
		"kapp-controller",
		"kapp-controller-packaging-global",
	}

	for _, ns := range namespaces {
		fmt.Printf("🔥 Deleting namespace %s\n", ns)

		_ = exec.Command("kubectl", "delete", "namespace", ns, "--ignore-not-found").Run()

		// hard finalize
		exec.Command("bash", "-c", fmt.Sprintf(`
kubectl get ns %s -o json 2>/dev/null \
| jq '.spec.finalizers=[]' \
| kubectl replace --raw /api/v1/namespaces/%s/finalize -f - 2>/dev/null || true
`, ns, ns)).Run()
	}

	return nil
}
