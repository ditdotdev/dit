package commands

import (
	"github.com/spf13/cobra"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull [REPOSITORY]",
	Short: "Pull a new data state from remote",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Pull(repo, guid, remote, tags, updateOnly)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit GUID to pull from, defaults to latest")
	pullCmd.Flags().StringVarP(&remote, "remote", "r", "", "name of the remote provider, defaults to origin")
	pullCmd.Flags().BoolVarP(&updateOnly, "update-only", "u", false, "update tags only, do not pull data")
	pullCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "filter commits to select commit to pull")
	pullCmd.Flags().SortFlags = false //TODO review flag sorting
}
