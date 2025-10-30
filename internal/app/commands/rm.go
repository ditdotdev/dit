package commands

import (
	"datadatdat/internal/app/providers"
	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm [REPOSITORY]",
	Short: "Remove a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider, repo := providers.ByName(args[0])
		provider.Remove(repo, force)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "Destroy all repositories")
}
