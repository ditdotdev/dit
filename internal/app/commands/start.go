package commands

import (
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [REPOSITORY]",
	Short: "Start a container for a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Start(repo)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
