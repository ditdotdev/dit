package commands

import (
	"github.com/spf13/cobra"
	"datadatdat/internal/app/providers"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install datadatdat infrastructure",

	Run: func(cmd *cobra.Command, args []string) {
		registry, _ := cmd.Flags().GetString("registry")
		provider = providers.Create(context, contextType, providers.GetAvailablePort())
		provider.Install([]string{"registry=" + registry}, verbose)
		providers.AddProvider(provider)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().String("registry", "datadatdat", "Registry URL for datadatdat docker image, defaults to datadatdat")
	installCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output of Datadatdat Server installation steps.")
}
