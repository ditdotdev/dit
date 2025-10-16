package commands

import (
	"github.com/spf13/cobra"
)

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag [REPOSITORY]",
	Short: "Tag objects in datadatdat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Tag(repo, guid, tags)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit to checkout")
	tagCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to filter latest commit, if commit is not specified")
}
