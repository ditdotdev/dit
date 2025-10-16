package commands

import (
	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log [REPOSITORY]",
	Short: "List commits for a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Log(repo, tags)
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to set")
}
