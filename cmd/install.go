package cmd

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-cli/core"
)

// ---------------------------------------------------------------------------
// Install entry point
// ---------------------------------------------------------------------------

func InstallAll() error {
	fmt.Println("🚀 BlanketOps Environments Installer")
	fmt.Println("─────────────────────────")

	fmt.Println("📦 Installing Carvel Kapp...")
	if err := core.DependenciesInstall([]string{
		"dependencies/carvel/release.yaml",
	}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Argo Events...")
	if err := core.DependenciesInstall([]string{
		"dependencies/argoevents/manifest.yaml",
	}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Tekton Pipelines...")
	if err := core.DependenciesInstall([]string{
		"dependencies/tekton/tekton_pipelines.yaml",
	}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Tekton Dashboard...")
	if err := core.DependenciesInstall([]string{
		"dependencies/tekton/tekton_dashboard.yaml",
	}); err != nil {
		return err
	}

	fmt.Println("📦 Installing Shipwright Build...")
	if err := core.DependenciesInstall([]string{
		"dependencies/shipwright/shipwright_build.yaml",
	}); err != nil {
		return err
	}

	if err := core.RunShipwrightCertSetup(); err != nil {
		return fmt.Errorf("Shipwright cert setup failed: %w", err)
	}

	// if err := core.InstallCrossplane(); err != nil {
	// 	return fmt.Errorf("crossplane setup failed: %w", err)
	// }

	// if err := core.InstallExternalSecrets(); err != nil {
	// 	return fmt.Errorf("externalsecrets setup failed: %w", err)
	// }

	fmt.Println("📦 Installing Build Strategies...")
	if err := core.DependenciesInstall([]string{
		"dependencies/cluster_strategies/buildpacks_v3.yaml",
		"dependencies/cluster_strategies/kaniko.yaml",
		"dependencies/cluster_strategies/buildah_shipwright_managed_push_cr.yaml",
	}); err != nil {
		return err
	}

	if err := core.InstallFluxCRDs(); err != nil {
		return err
	}

	fmt.Println("📦 Installing Flux UI + Config...")
	if err := core.DependenciesInstall([]string{
		"dependencies/flux/fluxcdui.yaml",
	}); err != nil {
		return err
	}

	// fmt.Println("📦 Installing Crossplane Github Upjet Provider...")
	// if err := core.DependenciesInstall([]string{
	// 	"dependencies/crossplane/provider.yaml",
	// }); err != nil {
	// 	return err
	// }

	if err := core.RunShipwrightCertSetup(); err != nil {
		return fmt.Errorf("shipwright certs setup failed: %w", err)
	}


	fmt.Println("🎉 BlanketOps environment installed successfully!")
	return nil
}
