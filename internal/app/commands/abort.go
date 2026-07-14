// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"github.com/spf13/cobra"
)

// abortCmd represents the abort command
var abortCmd = &cobra.Command{
	Use:   "abort [REPOSITORY]",
	Short: "Abort current push or pull operation",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Abort(repo)
	},
}

func init() {
	rootCmd.AddCommand(abortCmd)
	abortCmd.PersistentFlags()
}
