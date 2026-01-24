package cmd

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-cli/core"
)

func UninstallAll(paths []string) error {
	fmt.Println("🗑 BlanketOps Environments Uninstaller")
	fmt.Println("─────────────────────────")

	if err := core.RunUninstallScripts(); err != nil {
		return err
	}

	if err := core.UninstallManifests(paths); err != nil {
		return err
	}

	if err := core.UninstallClusterResources(); err != nil {
		return err
	}

	fmt.Println("✔ BlanketOps Environments uninstall complete")
	return nil
}
