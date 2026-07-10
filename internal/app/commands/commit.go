// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"github.com/spf13/cobra"
)

var message string

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit [REPOSITORY]",
	Short: "Commit current data state",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Commit(repo, message, tags)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringVarP(&message, "message", "m", "", "commit message")
	commitCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to set")
}
