package commands

import (
	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout [REPOSITORY]",
	Short: "Checkout a specific commit",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Checkout(repo, guid, tags)
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
	checkoutCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit to checkout")
	checkoutCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to filter latest commit, if commit is not specified")
}
