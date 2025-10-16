package commands

import (
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [REPOSITORY]",
	Short: "Delete objects from datadatdat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Delete(repo, guid, tags)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit GUID to delete")
	deleteCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tags to remove from a commit")
}
