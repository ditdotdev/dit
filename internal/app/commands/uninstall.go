package commands

import (
	"datadatdat/internal/app/providers"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall datadatdat infrastructure",

	Run: func(cmd *cobra.Command, args []string) {
		provider.Uninstall(force, removeImages)
		providers.Remove(provider.GetName())
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&force, "force", "f", false, "destroy all repositories")
	uninstallCmd.Flags().BoolVar(&removeImages, "remove-images", false, "remove datadatdat docker images")
}
