// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"github.com/ditdotdev/dit/internal/app/providers"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   subcmdInstall,
	Short: "Install dit infrastructure",

	Run: func(cmd *cobra.Command, args []string) {
		registry, _ := cmd.Flags().GetString("registry")
		provider = providers.Create(context, contextType, providers.GetAvailablePort())
		provider.Install([]string{"registry=" + registry}, verbose)
		providers.AddProvider(provider)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().String("registry", "ditdotdev", "Registry URL for dit docker image, defaults to ditdotdev")
	installCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output of Dit Server installation steps.")
}
